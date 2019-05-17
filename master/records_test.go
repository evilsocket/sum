package master

import (
	"context"
	pb "github.com/evilsocket/sum/proto"
	. "github.com/stretchr/testify/require"
	"strconv"
	"sync"
	"testing"
)

const numRoutines = 1024

func TestConcurrentCreateRecords(t *testing.T) {
	ns, err := setupNetwork(2, 1)
	Nil(t, err)
	defer cleanupNetwork(&ns)

	wg := sync.WaitGroup{}
	wg.Add(numRoutines)

	ms := ns.orchestrators[0].svc

	for i := 0; i < numRoutines; i++ {
		go func() {
			var resp *pb.RecordResponse
			var err error

			NotPanics(t, func() {
				resp, err = ms.CreateRecord(context.TODO(), &testRecord)
			})

			Nil(t, err)
			True(t, resp.Success, resp.Msg)
			wg.Done()
		}()
	}

	wg.Wait()

	Equal(t, numRoutines, ms.NumRecords())
}

func TestConcurrentDeleteRecords(t *testing.T) {
	ns, err := setupPopulatedNetwork(2, 1)
	Nil(t, err)
	defer cleanupNetwork(&ns)

	ms := ns.orchestrators[0].svc

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

				NotPanics(t, func() {
					resp, err = ms.DeleteRecord(context.TODO(), &pb.ById{Id: id})
				})

				Nil(t, err)
				True(t, resp.Success, resp.Msg)
			}
			wg.Done()
		}()
	}

	wg.Wait()

	Zero(t, ms.NumRecords())
}

func TestConcurrentCreateAndDelete(t *testing.T) {
	ns, err := setupNetwork(2, 1)
	Nil(t, err)
	defer cleanupNetwork(&ns)

	ms := ns.orchestrators[0].svc
	inputCh := genBenchRecords()
	idCh := make(chan uint64)
	wg := sync.WaitGroup{}
	wg1 := sync.WaitGroup{}
	wg.Add(numRoutines)
	wg1.Add(numRoutines)

	// create records

	for i := 0; i < numRoutines; i++ {
		go func() {
			for r := range inputCh {
				var resp *pb.RecordResponse
				var err error
				var id uint64

				resp, err = ms.CreateRecord(context.TODO(), r)
				Nil(t, err)
				True(t, resp.Success, resp.Msg)

				id, err = strconv.ParseUint(resp.Msg, 10, 64)
				Nil(t, err)

				idCh <- id
			}
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(idCh)
	}()

	// delete them

	for i := 0; i < numRoutines; i++ {
		go func() {
			for id := range idCh {
				var resp *pb.RecordResponse
				var err error

				NotPanics(t, func() {
					resp, err = ms.DeleteRecord(context.TODO(), &pb.ById{Id: id})
				})

				Nil(t, err)
				True(t, resp.Success, resp.Msg)
			}
			wg1.Done()
		}()
	}

	wg.Wait()
	wg1.Wait()

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
