package master

import (
	"context"
	pb "github.com/evilsocket/sum/proto"
	. "github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"io/ioutil"
	"testing"
)

func TestServiceCreateDuplicateOracle(t *testing.T) {
	setupFolders(t)
	defer teardown(t)

	if svc, err := NewClient(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.CreateOracle(context.TODO(), &testOracle); err != nil {
		t.Fatal(err)
	} else if !resp.Success {
		t.Fatalf("expected success response: %v", resp)
	} else if resp.Oracle != nil {
		t.Fatalf("unexpected oracle: %v", resp.Oracle)
	} else if resp.Msg != "1" {
		t.Fatalf("unexpected response message: %s", resp.Msg)
	} else if resp, err = svc.CreateOracle(context.TODO(), &testOracle); err != nil {
		t.Fatal(err)
	} else if resp.Success {
		t.Fatalf("expected unsuccessful response")
	} else if resp.Msg != "This oracle already exists." {
		t.Fatalf("unexpected response message: %s", resp.Msg)
	}
}

type faultyOracleClient struct {
	pb.SumServiceClient
	fail map[string]bool
}

func (c *faultyOracleClient) ReadOracle(ctx context.Context, in *pb.ById, opts ...grpc.CallOption) (*pb.OracleResponse, error) {
	if c.fail["read"] {
		return errOracleResponse("What?"), nil
	}
	return c.SumServiceClient.ReadOracle(ctx, in, opts...)
}
func (c *faultyOracleClient) DeleteOracle(ctx context.Context, in *pb.ById, opts ...grpc.CallOption) (*pb.OracleResponse, error) {
	if c.fail["delete"] {
		return errOracleResponse("What?"), nil
	}
	return c.SumServiceClient.DeleteOracle(ctx, in, opts...)
}

func TestService_StealOracle(t *testing.T) {
	ns, err := setupPopulatedNetwork(1, 1)
	NoError(t, err)
	defer cleanupNetwork(&ns)

	ns.orchestrators[0].updaterCancelFunc() // disable auto update

	ms := ns.orchestrators[0].svc
	sum := ns.nodes[0].svc

	resp, err := sum.CreateOracle(context.TODO(), &testOracle)
	NoError(t, err)
	True(t, resp.Success, resp.Msg)

	mockClient := &faultyOracleClient{
		SumServiceClient: ms.nodes[0].Client,
		fail:             make(map[string]bool),
	}
	ms.nodes[0].Client = mockClient

	ms.UpdateNodes()

	t.Run("read failure", func(t *testing.T) {
		log, restore := captureEvilsocketLog(t)
		defer restore()

		mockClient.fail["read"] = true
		defer delete(mockClient.fail, "read")

		ms.stealOracles()

		content, err := ioutil.ReadFile(log)
		NoError(t, err)
		Contains(t, string(content), "unable to read oracle #")

		Equal(t, 1, sum.NumOracles())
		Equal(t, 1, ms.NumOracles())
	})

	t.Run("load failure", func(t *testing.T) {
		defer func(oldId uint64) {
			ms.nextRaccoonId = oldId
		}(ms.nextRaccoonId)
		ms.nextRaccoonId = ms.nextRaccoonId - 1

		log, restore := captureEvilsocketLog(t)
		defer restore()

		ms.stealOracles()

		content, err := ioutil.ReadFile(log)
		NoError(t, err)
		Contains(t, string(content), "unable to load oracle #")

		Equal(t, 1, sum.NumOracles())
		Equal(t, 1, ms.NumOracles())
	})

	t.Run("delete failure", func(t *testing.T) {
		log, restore := captureEvilsocketLog(t)
		defer restore()

		mockClient.fail["delete"] = true
		defer delete(mockClient.fail, "delete")

		ms.stealOracles()

		content, err := ioutil.ReadFile(log)
		NoError(t, err)
		Contains(t, string(content), "cannot delete oracle #")

		Equal(t, 1, sum.NumOracles())
		Equal(t, 2, ms.NumOracles())

		resp, err := ms.DeleteOracle(context.TODO(), &pb.ById{Id: 2})
		NoError(t, err)
		True(t, resp.Success, resp.Msg)
	})
}
