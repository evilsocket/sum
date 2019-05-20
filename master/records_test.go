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
