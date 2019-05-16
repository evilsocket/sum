package master

import (
	"context"
	"fmt"
	pb "github.com/evilsocket/sum/proto"
	. "github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func TestBalancing(t *testing.T) {
	SetCommunicationTimeout(time.Minute)

	dir1, err := setupEmptyTmpFolder()
	Nil(t, err)
	defer os.RemoveAll(dir1)
	dir2, err := setupEmptyTmpFolder()
	Nil(t, err)
	defer os.RemoveAll(dir2)

	node1, sum1 := spawnNode(t, 12345, dir1)
	defer node1.Stop()
	node2, sum2 := spawnNode(t, 12346, dir2)
	defer node2.Stop()

	testRecord := &pb.Record{Data: []float32{0.1, 0.2, 0.3}, Meta: map[string]string{"name": "test"}}

	for i := 1; i <= 100; i++ {
		rCopy := &pb.Record{Id: uint64(i), Data: testRecord.Data, Meta: testRecord.Meta}
		resp, err := sum1.CreateRecordWithId(context.Background(), rCopy)
		Nil(t, err)
		True(t, resp.Success)
	}

	for i := 101; i <= 120; i++ {
		rCopy := &pb.Record{Id: uint64(i), Data: testRecord.Data, Meta: testRecord.Meta}
		resp, err := sum2.CreateRecordWithId(context.Background(), rCopy)
		Nil(t, err)
		True(t, resp.Success)
	}

	// now we have 100 records on node1, 20 on node2

	master, _ := spawnOrchestrator(t, 12347, "localhost:12345,localhost:12346")
	defer master.Stop()

	Equal(t, sum1.NumRecords(), sum2.NumRecords())
}

// balance multiple networks

func TestBalanceMultipleNetworks(t *testing.T) {
	SetCommunicationTimeout(time.Hour)
	ns, err := setupNetwork(6, 2)
	NoError(t, err)
	defer cleanupNetwork(&ns)

	ms, ms1 := ns.orchestrators[0].svc, ns.orchestrators[1].svc

	slave1, slave2, slave3 := ns.nodes[0].svc, ns.nodes[1].svc, ns.nodes[2].svc
	slave4, slave5, slave6 := ns.nodes[3].svc, ns.nodes[4].svc, ns.nodes[5].svc

	for i := 1; i <= 2; i++ {
		resp, err := ms1.DeleteNode(context.TODO(), &pb.ById{Id: uint64(i)})
		NoError(t, err)
		True(t, resp.Success)
	}

	for i := 3; i <= 6; i++ {
		resp, err := ms.DeleteNode(context.TODO(), &pb.ById{Id: uint64(i)})
		NoError(t, err)
		True(t, resp.Success)
	}

	Equal(t, 2, len(ms.nodes))
	Equal(t, 4, len(ms1.nodes))

	// now we have 2 different networks: 2 nodes managed by one orchestrator and another 4 managed by another one

	nodes := fmt.Sprintf("%s,%s", ms.address, ms1.address)

	server, ms2 := spawnOrchestrator(t, 10000, nodes)
	defer server.Stop()

	arg := &pb.Records{}

	for i := 0; i < 80; i++ {
		rCopy := &pb.Record{Id: uint64(i + 1), Meta: testRecord.Meta, Data: testRecord.Data}
		arg.Records = append(arg.Records, rCopy)
	}

	resp, err := ms2.CreateRecordsWithId(context.TODO(), arg)
	NoError(t, err)
	True(t, resp.Success)

	Equal(t, 80, ms2.NumRecords())
	Equal(t, ms1.NumRecords(), ms.NumRecords())

	Equal(t, slave1.NumRecords(), slave2.NumRecords())

	Equal(t, slave3.NumRecords(), slave4.NumRecords())
	Equal(t, slave4.NumRecords(), slave5.NumRecords())
	Equal(t, slave5.NumRecords(), slave6.NumRecords())
}
