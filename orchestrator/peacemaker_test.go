package main

import (
	"context"
	"fmt"
	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/service"
	. "github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPeacemaker(t *testing.T) {
	newTimeout := time.Minute
	timeout = &newTimeout
	pollPeriod = &newTimeout

	dir1, err := ioutil.TempDir("", "")
	Nil(t, err)
	defer os.RemoveAll(dir1)
	dir2, err := ioutil.TempDir("", "")
	Nil(t, err)
	defer os.RemoveAll(dir2)

	for _, baseDir := range []string{dir1, dir2} {
		for _, childDir := range []string{"data", "oracles"} {
			err = os.Mkdir(filepath.Join(baseDir, childDir), 0755)
			Nil(t, err)
		}
	}

	node1, sum1 := spawnNode(t, 12345, dir1)
	defer node1.Stop()
	node2, sum2 := spawnNode(t, 12346, dir2)
	defer node2.Stop()

	testRecord := &pb.Record{Id: 1, Data: []float32{0.1, 0.2, 0.3}, Meta: map[string]string{"name": "test"}}
	testRecord1 := &pb.Record{Id: 2, Data: []float32{0.1, 0.2, 0.3}, Meta: map[string]string{"name": "test1"}}
	testRecord2 := &pb.Record{Id: 2, Data: []float32{0.2, 0.4, 0.6}, Meta: map[string]string{"name": "test2"}}

	toCreate := map[*service.Service][]*pb.Record{
		sum1: {testRecord, testRecord1},
		sum2: {testRecord, testRecord2},
	}

	for svc, records := range toCreate {
		for _, r := range records {
			rCopy := &pb.Record{Id: r.Id, Data: r.Data, Meta: r.Meta}
			resp, err := svc.CreateRecordWithId(context.Background(), rCopy)
			Nil(t, err)
			True(t, resp.Success, resp.Msg)
		}
	}

	// now we have 2 nodes, both with the same record #1, but with a record #2 which is different

	master, _ := spawnOrchestrator(t, 12347, "localhost:12345,localhost:12346")
	defer master.Stop()

	recName2record := make(map[string]*pb.Record)

	for _, svc := range []*service.Service{sum1, sum2} {
		resp, err := svc.ListRecords(context.Background(), &pb.ListRequest{Page: 1, PerPage: 1024})
		Nil(t, err)
		for _, r := range resp.Records {
			name := r.Meta["name"]
			_, exists := recName2record[name]
			False(t, exists, fmt.Sprintf("Record %d found multiple time", r.Id))
			recName2record[name] = r
		}
	}

	// check that records exist with the same data

	r, exist := recName2record["test"]
	True(t, exist)
	Equal(t, testRecord.Data, r.Data)

	r, exist = recName2record["test1"]
	True(t, exist)
	Equal(t, testRecord1.Data, r.Data)

	r, exist = recName2record["test2"]
	True(t, exist)
	Equal(t, testRecord2.Data, r.Data)

}
