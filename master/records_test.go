package master

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	pb "github.com/evilsocket/sum/proto"
	"github.com/stretchr/testify/assert"
	. "github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const numRoutines = 1024

func TestService_CreateRecord_NoNodes(t *testing.T) {
	ns, err := setupNetwork(0, 1)
	NoError(t, err)
	defer cleanupNetwork(&ns)

	resp, err := ns.orchestrators[0].svc.CreateRecord(context.TODO(), &testRecord)
	NoError(t, err)
	False(t, resp.Success)
	Equal(t, "No nodes available, try later", resp.Msg)
}

func TestService_UpdateRecord_ConnectionError(t *testing.T) {
	ns, err := setupPopulatedNetwork(1, 1)
	NoError(t, err)
	defer cleanupNetwork(&ns)

	ns.nodes[0].server.Stop()

	resp, err := ns.orchestrators[0].svc.UpdateRecord(context.TODO(), &testRecord)
	NoError(t, err)
	False(t, resp.Success)

	errMsgRgx := `^No node was able to satisfy your request: \[node 1: rpc error: code = Unavailable`

	Regexp(t, errMsgRgx, resp.Msg)
}

func TestService_MultipleAnswers(t *testing.T) {
	ns, err := setupPopulatedNetwork(2, 1)
	NoError(t, err)
	defer cleanupNetwork(&ns)

	// duplicate a record on one node

	sum1 := ns.nodes[0].svc
	sum2 := ns.nodes[1].svc

	resp, err := sum1.ListRecords(context.TODO(), &pb.ListRequest{PerPage: 1, Page: 1})
	NoError(t, err)
	Equal(t, 1, len(resp.Records))

	// deep copy the record to avoid having the nodes
	// sharing the same record instance ( since we are not using the HTTP server )
	aRecorBytes, err := json.Marshal(resp.Records[0])
	NoError(t, err)
	aRecord := &pb.Record{}
	NoError(t, json.Unmarshal(aRecorBytes, aRecord))

	resp1, err := sum2.CreateRecordWithId(context.TODO(), aRecord)
	NoError(t, err)
	True(t, resp1.Success, resp1.Msg)

	newLog, restoreLog := captureEvilsocketLog(t)
	defer os.Remove(newLog)
	defer restoreLog()

	// wait all the parallel requests to complete
	commContextIsCancellable = false
	defer func() { commContextIsCancellable = true }()

	// setup complete

	ms := ns.orchestrators[0].svc

	// read

	resp1, err = ms.ReadRecord(context.TODO(), &pb.ById{Id: aRecord.Id})
	NoError(t, err)
	True(t, resp1.Success, resp1.Msg)

	// update

	aRecord.Data[0] = aRecord.Data[0] / 2
	aRecord.Meta = map[string]string{"updated": "true"}

	resp1, err = ms.UpdateRecord(context.TODO(), aRecord)
	NoError(t, err)
	True(t, resp1.Success, resp1.Msg)

	// delete

	resp1, err = ms.DeleteRecord(context.TODO(), &pb.ById{Id: aRecord.Id})
	NoError(t, err)
	True(t, resp1.Success, resp1.Msg)

	restoreLog()

	logContent, err := ioutil.ReadFile(newLog)
	found := 0

	scanner := bufio.NewScanner(strings.NewReader(string(logContent)))
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "Got 2 results when only one was expected") {
			found++
		}
	}

	Equalf(t, 3, found, "Expected 3 lines of log to match, got %d", found)
}

func TestConcurrentCreateRecords(t *testing.T) {
	ns, err := setupNetwork(2, 1)
	Nil(t, err)
	defer cleanupNetwork(&ns)

	wg := sync.WaitGroup{}
	wg.Add(numRoutines)

	failures := uint64(0)
	ms := ns.orchestrators[0].svc

	for i := 0; i < numRoutines; i++ {
		go func() {
			var resp *pb.RecordResponse
			var err error

			if !assert.NotPanics(t, func() {
				resp, err = ms.CreateRecord(context.TODO(), &testRecord)
			}) ||
				!assert.Nil(t, err) ||
				!assert.True(t, resp.Success, resp.Msg) {
				atomic.AddUint64(&failures, 1)
			}
			wg.Done()
		}()
	}

	wg.Wait()

	if failures > 0 {
		t.Fatalf("%d routines failed", failures)
	}

	Equal(t, numRoutines, ms.NumRecords())
}

func TestConcurrentDeleteRecords(t *testing.T) {
	ns, err := setupPopulatedNetwork(2, 1)
	Nil(t, err)
	defer cleanupNetwork(&ns)

	ms := ns.orchestrators[0].svc
	failures := uint64(0)

	idCh := make(chan uint64)
	wg := sync.WaitGroup{}
	wg.Add(numRoutines)

	go func() {
		for i := 1; i <= numBenchRecords; i++ {
			idCh <- uint64(i)
		}
		close(idCh)
	}()

	for i := 0; i < numRoutines; i++ {
		go func() {
			for id := range idCh {
				var resp *pb.RecordResponse
				var err error

				if !assert.NotPanics(t, func() {
					resp, err = ms.DeleteRecord(context.TODO(), &pb.ById{Id: id})
				}) ||
					!assert.Nil(t, err) ||
					!assert.True(t, resp.Success, resp.Msg) {
					atomic.AddUint64(&failures, 1)
				}
			}
			wg.Done()
		}()
	}

	wg.Wait()

	if failures > 0 {
		t.Fatalf("%d routines failed", failures)
	}

	Zero(t, ms.NumRecords())
}

func TestConcurrentCreateAndDelete(t *testing.T) {
	ns, err := setupNetwork(2, 1)
	Nil(t, err)
	defer cleanupNetwork(&ns)

	ms := ns.orchestrators[0].svc
	inputCh := genBenchRecords()
	idCh := make(chan uint64)
	wg := &sync.WaitGroup{}
	wg1 := &sync.WaitGroup{}
	wg.Add(numRoutines)
	wg1.Add(numRoutines)

	failures := uint64(0)

	// create records

	for i := 0; i < numRoutines; i++ {
		go func() {
			for r := range inputCh {
				var resp *pb.RecordResponse
				var err error
				var id uint64

				if !assert.NotPanics(t, func() {
					resp, err = ms.CreateRecord(context.TODO(), r)
				}) ||
					!assert.NoError(t, err) ||
					!assert.True(t, resp.Success, resp.Msg) {
					atomic.AddUint64(&failures, 1)
					continue
				}

				id, err = strconv.ParseUint(resp.Msg, 10, 64)
				if !assert.NoError(t, err) {
					atomic.AddUint64(&failures, 1)
					continue
				}

				idCh <- id
			}
			wg.Done()
		}()
	}

	// delete them

	for i := 0; i < numRoutines; i++ {
		go func() {
			for id := range idCh {
				var resp *pb.RecordResponse
				var err error

				if !assert.NotPanics(t, func() {
					resp, err = ms.DeleteRecord(context.TODO(), &pb.ById{Id: id})
				}) ||
					!assert.NoError(t, err) ||
					!assert.True(t, resp.Success, resp.Msg) {
					atomic.AddUint64(&failures, 1)
				}
			}

			wg1.Done()
		}()
	}

	wg.Wait()
	close(idCh)
	wg1.Wait()

	if failures > 0 {
		t.Fatalf("%d routines failed", failures)
	}

	Zero(t, ms.NumRecords())
}

func TestMuxService_DeleteRecords(t *testing.T) {
	ns, err := setupPopulatedNetwork(2, 1)
	Nil(t, err)
	defer cleanupNetwork(&ns)

	arg := &pb.RecordIds{}

	ms := ns.orchestrators[0].svc
	node1 := ns.nodes[0].svc
	node2 := ns.nodes[1].svc

	for id := 1; id <= numBenchRecords; id++ {
		arg.Ids = append(arg.Ids, uint64(id))
	}

	resp, err := ms.DeleteRecords(context.TODO(), arg)
	NoError(t, err)
	True(t, resp.Success, resp.Msg)

	Zero(t, node1.NumRecords())
	Zero(t, node2.NumRecords())
	Zero(t, ms.NumRecords())
}

func TestMuxService_DeleteRecordsInvalid(t *testing.T) {
	ns, err := setupPopulatedNetwork(2, 1)
	Nil(t, err)
	defer cleanupNetwork(&ns)

	arg := &pb.RecordIds{}

	ms := ns.orchestrators[0].svc
	node1 := ns.nodes[0].svc
	node2 := ns.nodes[1].svc

	for id := 1; id <= numBenchRecords; id++ {
		arg.Ids = append(arg.Ids, uint64(id))
	}
	arg.Ids = append(arg.Ids, uint64(numBenchRecords+1))

	resp, err := ms.DeleteRecords(context.TODO(), arg)
	NoError(t, err)
	False(t, resp.Success)

	errMsg := fmt.Sprintf("%d", numBenchRecords)

	Equal(t, errMsg, resp.Msg)

	Zero(t, node1.NumRecords())
	Zero(t, node2.NumRecords())
	Zero(t, ms.NumRecords())
}

type oldVersionClient struct {
	pb.SumInternalServiceClient
	msg string
}

func (c *oldVersionClient) DeleteRecords(ctx context.Context, in *pb.RecordIds, opts ...grpc.CallOption) (*pb.RecordResponse, error) {
	resp, err := c.SumInternalServiceClient.DeleteRecords(ctx, in, opts...)
	resp.Msg = c.msg
	return resp, err
}

func TestMuxService_DeleteRecordsWrongAnswer(t *testing.T) {
	ns, err := setupPopulatedNetwork(2, 1)
	Nil(t, err)
	defer cleanupNetwork(&ns)

	arg := &pb.RecordIds{}

	ms := ns.orchestrators[0].svc
	node1 := ns.nodes[0].svc
	node2 := ns.nodes[1].svc

	for id := 1; id <= numBenchRecords; id++ {
		arg.Ids = append(arg.Ids, uint64(id))
	}

	originalClient := ms.nodes[0].InternalClient
	mockClient := &oldVersionClient{
		SumInternalServiceClient: originalClient,
		msg:                      "so number, much numeric, very arabic, wow",
	}
	ms.nodes[0].InternalClient = mockClient
	defer func() {
		ms.nodes[0].InternalClient = originalClient
	}()

	resp, err := ms.DeleteRecords(context.TODO(), arg)
	NoError(t, err)
	False(t, resp.Success)

	errMsg := fmt.Sprintf("unable to parse node '%v' response '%v' as uint: ", ms.nodes[0].Name, mockClient.msg)

	Contains(t, resp.Msg, errMsg)

	ms.UpdateNodes()

	Zero(t, node1.NumRecords())
	Zero(t, node2.NumRecords())
	Zero(t, ms.NumRecords())
}

func TestService_ReadRecord_ConnErr(t *testing.T) {
	ns, err := setupPopulatedNetwork(1, 1)
	Nil(t, err)
	defer cleanupNetwork(&ns)

	ns.nodes[0].server.Stop()

	resp, err := ns.orchestrators[0].svc.ReadRecord(context.TODO(), &pb.ById{Id: 1})
	NoError(t, err)
	False(t, resp.Success)

	errRgx := `^No node was able to satisfy your request: \[node 1: rpc error: code = Unavailable`

	Regexp(t, errRgx, resp.Msg)

}

func TestServiceListRecordsMultiPageMultiNode(t *testing.T) {
	ns, err := setupPopulatedNetwork(2, 1)
	NoError(t, err)
	defer cleanupNetwork(&ns)

	t.Run("WithPageLessThan1Node", func(t *testing.T) {
		list := pb.ListRequest{
			Page:    1,
			PerPage: 2,
		}

		ms := ns.orchestrators[0].svc
		numPages := uint64(numBenchRecords) / 2
		if numBenchRecords%2 != 0 {
			numPages++
		}

		if resp, err := ms.ListRecords(context.TODO(), &list); err != nil {
			t.Fatal(err)
		} else if resp.Total != numBenchRecords {
			t.Fatalf("expected %d total records, got %d", testRecords, resp.Total)
		} else if resp.Pages != numPages {
			t.Fatalf("expected %d pages got %d", numPages, resp.Pages)
		} else if len(resp.Records) != 2 {
			t.Fatalf("expected %d total records, got %d", 2, len(resp.Records))
		}
	})

	t.Run("WithPageMoreTHan1Node", func(t *testing.T) {
		list := pb.ListRequest{
			Page:    1,
			PerPage: (numBenchRecords / 2) + 1,
		}

		ms := ns.orchestrators[0].svc
		numPages := numBenchRecords / list.PerPage
		if numBenchRecords%list.PerPage != 0 {
			numPages++
		}

		if resp, err := ms.ListRecords(context.TODO(), &list); err != nil {
			t.Fatal(err)
		} else if resp.Total != numBenchRecords {
			t.Fatalf("expected %d total records, got %d", testRecords, resp.Total)
		} else if resp.Pages != numPages {
			t.Fatalf("expected %d pages got %d", numPages, resp.Pages)
		} else if uint64(len(resp.Records)) != list.PerPage {
			t.Fatalf("expected %d total records, got %d", list.PerPage, len(resp.Records))
		}
	})

}

func TestServiceListRecordsZeroPerPage(t *testing.T) {
	ns, err := setupPopulatedNetwork(2, 1)
	NoError(t, err)
	defer cleanupNetwork(&ns)

	list := pb.ListRequest{
		Page:    1,
		PerPage: 0,
	}

	ms := ns.orchestrators[0].svc
	numPages := uint64(numBenchRecords)

	if resp, err := ms.ListRecords(context.TODO(), &list); err != nil {
		t.Fatal(err)
	} else if resp.Total != numBenchRecords {
		t.Fatalf("expected %d total records, got %d", testRecords, resp.Total)
	} else if resp.Pages != numPages {
		t.Fatalf("expected %d pages got %d", numPages, resp.Pages)
	} else if len(resp.Records) != 1 {
		t.Fatalf("expected %d total records, got %d", 1, len(resp.Records))
	}
}

func TestService_ListRecords(t *testing.T) {
	ns, err := setupPopulatedNetwork(4, 1)
	NoError(t, err)
	defer cleanupNetwork(&ns)

	ms := ns.orchestrators[0].svc

	testList := func(t *testing.T, expectedNumPages, page, perPage uint64) {
		list := pb.ListRequest{Page: page, PerPage: perPage}
		if resp, err := ms.ListRecords(context.TODO(), &list); err != nil {
			t.Fatal(err)
		} else if resp.Total != numBenchRecords {
			t.Fatalf("expected %d total records, got %d", testRecords, resp.Total)
		} else if resp.Pages != expectedNumPages {
			t.Fatalf("expected %d pages got %d", expectedNumPages, resp.Pages)
		} else if len(resp.Records) != int(perPage) {
			t.Fatalf("expected %d total records, got %d", perPage, len(resp.Records))
		}
	}

	t.Run("Strictly contained in node", func(t *testing.T) {
		testList(t, 16, 2, numBenchRecords/16)
	})

	t.Run("At the beginning of node", func(t *testing.T) {
		testList(t, 6, 1, numBenchRecords/5)
	})

	t.Run("Across nodes", func(t *testing.T) {
		testList(t, 6, 2, numBenchRecords/5)
	})

	t.Run("At the end of node", func(t *testing.T) {
		testList(t, 8, 2, numBenchRecords/8)
	})

	t.Run("With an entire node", func(t *testing.T) {
		testList(t, 2, 1, numBenchRecords/2+2)
	})
}

func TestServiceListRecordsSinglePage_ConnErr(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	list := pb.ListRequest{
		Page:    0,
		PerPage: testRecords,
	}

	if _, err := NewClient(testFolder); err != nil {
		t.Fatal(err)
	}

	ms := network.orchestrators[0].svc
	network.nodes[0].server.Stop()

	if resp, err := ms.ListRecords(context.TODO(), &list); err != nil {
		t.Fatal(err)
	} else if resp.Total != testRecords {
		t.Fatalf("expected %d total records, got %d", testRecords, resp.Total)
	} else if resp.Pages != 1 {
		t.Fatalf("expected 1 page, got %d", resp.Pages)
	} else if len(resp.Records) != 0 {
		t.Fatalf("expected 0 total records, got %d", len(resp.Records))
	}
}

type clientFailureProxy struct {
	pb.SumInternalServiceClient
	failNextCall bool
	errorString  string
}

func (cfp *clientFailureProxy) CreateRecordsWithId(ctx context.Context, arg *pb.Records, options ...grpc.CallOption) (*pb.RecordResponse, error) {
	if !cfp.failNextCall {
		cfp.failNextCall = true
		return cfp.SumInternalServiceClient.CreateRecordsWithId(ctx, arg, options...)
	}
	return nil, fmt.Errorf("%s", cfp.errorString)
}

func TestService_CreateRecordsWithId2(t *testing.T) {
	ns, err := setupNetwork(2, 1)
	NoError(t, err)
	defer cleanupNetwork(&ns)

	ms := ns.orchestrators[0].svc

	Nrecs := 5
	recs := make([]*pb.Record, 0, Nrecs)
	for i := 0; i < Nrecs; i++ {
		rCopy := *&testRecord
		rCopy.Id = uint64(i + 1)
		recs = append(recs, &rCopy)
	}

	t.Run("WithoutNode", func(t *testing.T) {
		ns.nodes[1].server.Stop()

		resp, err := ms.CreateRecordsWithId(context.TODO(), &pb.Records{Records: recs})

		NoError(t, err)
		True(t, resp.Success, resp.Msg)
	})

	t.Run("Rollback", func(t *testing.T) {
		oldClient := ms.nodes[0].InternalClient
		ms.nodes[0].InternalClient = &clientFailureProxy{
			SumInternalServiceClient: oldClient,
			errorString:              "FAIL",
		}
		defer func() {
			ms.nodes[0].InternalClient = oldClient
		}()

		resp, err := ms.CreateRecordsWithId(context.TODO(), &pb.Records{Records: recs})

		NoError(t, err)
		False(t, resp.Success)

		Equal(t, `Unable to create records on fallback node 1: FAIL`, resp.Msg)
	})

	t.Run("WithoutNodes", func(t *testing.T) {
		ns.nodes[0].server.Stop()

		resp, err := ms.CreateRecordsWithId(context.TODO(), &pb.Records{Records: recs})

		NoError(t, err)
		False(t, resp.Success)

		rgx := `^Cannot create records on nodes: last error = rpc error: code = Unavailable`

		Regexp(t, rgx, resp.Msg)
	})
}

func TestService_CreateRecordsWithId3(t *testing.T) {
	ns, err := setupNetwork(0, 1)
	NoError(t, err)
	defer cleanupNetwork(&ns)

	ms := ns.orchestrators[0].svc
	resp, err := ms.CreateRecordsWithId(context.TODO(), &pb.Records{})

	NoError(t, err)
	False(t, resp.Success)
	Equal(t, "No nodes available, try later", resp.Msg)
}

func TestService_CreateRecordWithId2(t *testing.T) {
	ns, err := setupNetwork(0, 1)
	NoError(t, err)
	defer cleanupNetwork(&ns)

	ms := ns.orchestrators[0].svc
	resp, err := ms.CreateRecordWithId(context.TODO(), &testRecord)

	NoError(t, err)
	False(t, resp.Success)
	Equal(t, "No nodes available, try later", resp.Msg)
}

type clientListDelayProxy struct {
	pb.SumServiceClient
	delay time.Duration
}

func (p *clientListDelayProxy) ListRecords(ctx context.Context, arg *pb.ListRequest, opts ...grpc.CallOption) (*pb.RecordListResponse, error) {
	time.Sleep(p.delay)
	return p.SumServiceClient.ListRecords(ctx, arg, opts...)
}

func TestService_ListCancel(t *testing.T) {
	ns, err := setupNetwork(1, 1)
	NoError(t, err)
	defer cleanupNetwork(&ns)

	ms := ns.orchestrators[0].svc

	resp, err := ms.CreateRecord(context.TODO(), &testRecord)
	NoError(t, err)
	True(t, resp.Success)

	originalClient := ms.nodes[0].Client

	ms.nodes[0].Client = &clientListDelayProxy{
		SumServiceClient: originalClient,
		delay:            400 * time.Millisecond,
	}
	// do not restore the client as it creates a data race condition

	ctx, _ := context.WithTimeout(context.TODO(), 300*time.Millisecond)

	resp1, err := ms.ListRecords(ctx, &pb.ListRequest{PerPage: 1, Page: 1})
	NoError(t, err)
	Equal(t, 1, int(resp1.Total))
	Equal(t, 1, int(resp1.Pages))
	Empty(t, resp1.Records)
}

func TestService_CreateRecords(t *testing.T) {
	ns, err := setupNetwork(1, 1)
	NoError(t, err)
	defer cleanupNetwork(&ns)

	ms := ns.orchestrators[0].svc

	t.Run("OK", func(t *testing.T) {
		records := []*pb.Record{
			genBenchRecord(), genBenchRecord(),
		}

		prevNum := ms.NumRecords()

		resp, err := ms.CreateRecords(context.TODO(), &pb.Records{Records: records})
		NoError(t, err)
		True(t, resp.Success, resp.Msg)

		Equal(t, prevNum+2, ms.NumRecords())
	})

	t.Run("ClientError", func(t *testing.T) {
		records := []*pb.Record{
			genBenchRecord(), genBenchRecord(),
		}

		mockedClient := &clientFailureProxy{
			errorString: "wow", failNextCall: true,
		}

		defer func(originalClient pb.SumInternalServiceClient) {
			ms.nodes[0].InternalClient = originalClient
		}(ms.nodes[0].InternalClient)

		ms.nodes[0].InternalClient = mockedClient

		prevNum := ms.NumRecords()

		resp, err := ms.CreateRecords(context.TODO(), &pb.Records{Records: records})
		NoError(t, err)
		False(t, resp.Success)

		Equal(t, prevNum, ms.NumRecords())
	})
}

func TestService_CreateRecords_NoNodes(t *testing.T) {
	ns, err := setupNetwork(0, 1)
	NoError(t, err)
	defer cleanupNetwork(&ns)

	records := []*pb.Record{&testRecord}
	resp, err := ns.orchestrators[0].svc.CreateRecords(context.TODO(), &pb.Records{Records: records})
	NoError(t, err)
	False(t, resp.Success)
	Contains(t, resp.Msg, "No nodes available")
}

func TestService_CreateRecords_Node(t *testing.T) {
	ns, err := setupNetwork(1, 0)
	NoError(t, err)
	defer cleanupNetwork(&ns)

	records := []*pb.Record{&testRecord}
	resp, err := ns.nodes[0].svc.CreateRecords(context.TODO(), &pb.Records{Records: records})
	NoError(t, err)
	True(t, resp.Success, resp.Msg)
	Equal(t, 1, ns.nodes[0].svc.NumRecords())
}
