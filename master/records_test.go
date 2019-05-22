package master

import (
	"context"
	pb "github.com/evilsocket/sum/proto"
	"github.com/stretchr/testify/assert"
	. "github.com/stretchr/testify/require"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
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
	True(t, resp.Success)

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
		ns.nodes[0].server.Stop()

		resp, err := ms.CreateRecordsWithId(context.TODO(), &pb.Records{Records: recs})

		NoError(t, err)
		True(t, resp.Success, resp.Msg)
	})

	t.Run("WithoutNodes", func(t *testing.T) {
		ns.nodes[1].server.Stop()

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
