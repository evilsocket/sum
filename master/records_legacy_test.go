package master

import (
	"context"
	"fmt"
	"github.com/evilsocket/sum/node/storage"
	pb "github.com/evilsocket/sum/proto"
	"github.com/golang/protobuf/proto"
	"regexp"
	"testing"
)

var (
	byID          = pb.ById{Id: 1}
	updatedRecord = pb.Record{
		Id:   555,
		Data: []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 666},
		Meta: map[string]string{
			"idk": "idk",
		},
		Shape: []uint64{10},
	}
)

func TestServiceErrRecordResponse(t *testing.T) {
	if r := errRecordResponse("test %d", 123); r.Success {
		t.Fatal("success should be false")
	} else if r.Msg != "test 123" {
		t.Fatalf("unexpected message: %s", r.Msg)
	} else if r.Record != nil {
		t.Fatalf("unexpected record pointer: %v", r.Record)
	}
}

func TestServiceCreateRecord(t *testing.T) {
	setupFolders(t)
	defer teardown(t)

	if svc, err := NewClient(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.CreateRecord(context.TODO(), &testRecord); err != nil {
		t.Fatal(err)
	} else if !resp.Success {
		t.Fatalf("expected success response: %v", resp)
	} else if resp.Record != nil {
		t.Fatalf("unexpected record pointer: %v", resp.Record)
	} else if resp.Msg != "1" {
		t.Fatalf("unexpected response message: %s", resp.Msg)
	}
}

func TestServiceCreateRecordNotUniqueId(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	svc, err := NewClient(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	// ok this is kinda cheating, but i want full coverage
	network.orchestrators[0].svc.nextId = 1
	if resp, err := svc.CreateRecord(context.TODO(), &testRecord); err != nil {
		t.Fatal(err)
	} else if resp.Success {
		t.Fatalf("expected error response: %v", resp)
	} else if resp.Record != nil {
		t.Fatalf("unexpected record pointer: %v", resp.Record)
	} else if resp.Msg != storage.ErrInvalidID.Error() {
		t.Fatalf("unexpected response message: %s", resp.Msg)
	}
}

func TestServiceUpdateRecord(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	svc, err := NewClient(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	updatedRecord.Id = 1
	if resp, err := svc.UpdateRecord(context.TODO(), &updatedRecord); err != nil {
		t.Fatal(err)
	} else if !resp.Success {
		t.Fatalf("expected success response: %v", resp)
	} else if resp.Record != nil {
		t.Fatalf("unexpected record pointer: %v", resp.Record)
	} else if stored, err := svc.ReadRecord(context.TODO(), &pb.ById{Id: updatedRecord.Id}); err != nil {
		t.Fatalf("unaexpected error %v", err)
	} else if stored.Record == nil {
		t.Fatal("expected stored record with id 1")
	} else if !proto.Equal(stored.Record, &updatedRecord) {
		t.Fatal("record has not been updated as expected")
	}
}

func TestService_FindRecords(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := NewClient(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.FindRecords(context.TODO(), &pb.ByMeta{Meta: "666", Value: "666"}); err != nil {
		t.Fatal(err)
	} else if !resp.Success {
		t.Fatalf("expected success response: %v", resp)
	} else if len(resp.Records) != testRecords {
		t.Fatalf("unexpected records: %v", resp.Records)
	}
}

func TestService_FindRecordsNotFound(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := NewClient(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.FindRecords(context.TODO(), &pb.ByMeta{Meta: "not", Value: "found"}); err != nil {
		t.Fatal(err)
	} else if !resp.Success {
		t.Fatalf("expected success response: %v", resp)
	} else if len(resp.Records) != 0 {
		t.Fatalf("unexpected records: %v", resp.Records)
	}
}

func TestService_FindRecordsErrors(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	var svc pb.SumServiceClient
	var err error

	if svc, err = NewClient(testFolder); err != nil {
		t.Fatal(err)
	}

	ni := network.orchestrators[0].svc.nodes[0]
	network.nodes[0].server.Stop() // trigger connection error

	rgx := regexp.MustCompile(fmt.Sprintf(`Errors from nodes: \[.*Error while dialing dial tcp %s: connect: connection refused"\]`, ni.Name))

	if resp, err := svc.FindRecords(context.TODO(), &pb.ByMeta{Meta: "not", Value: "found"}); err != nil {
		t.Fatal(err)
	} else if resp.Success {
		t.Fatalf("expected unsuccessful response: %v", resp)
	} else if len(resp.Records) != 0 {
		t.Fatalf("unexpected records: %v", resp.Records)
	} else if rgx.Find([]byte(resp.Msg)) == nil {
		t.Fatalf("unexpected error message: %v", resp.Msg)
	}
}

func TestServiceUpdateRecordWithInvalidId(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	svc, err := NewClient(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	updatedRecord.Id = 666
	if resp, err := svc.UpdateRecord(context.TODO(), &updatedRecord); err != nil {
		t.Fatal(err)
	} else if resp.Success {
		t.Fatalf("expected error response: %v", resp)
	} else if resp.Record != nil {
		t.Fatalf("unexpected record pointer: %v", resp.Record)
	} else if resp.Msg != storage.ErrRecordNotFound.Error() {
		t.Fatalf("unexpected response message: %s", resp.Msg)
	}
}

func TestServiceReadRecord(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := NewClient(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.ReadRecord(context.TODO(), &byID); err != nil {
		t.Fatal(err)
	} else if !resp.Success {
		t.Fatalf("expected success response: %v", resp)
	} else if resp.Record == nil {
		t.Fatal("expected record pointer")
	} else if testRecord.Id = byID.Id; !proto.Equal(resp.Record, &testRecord) {
		t.Fatalf("unexpected record: %v", resp.Record)
	}
}

func TestServiceReadRecordWithInvalidId(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := NewClient(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.ReadRecord(context.TODO(), &pb.ById{Id: 666}); err != nil {
		t.Fatal(err)
	} else if resp.Success {
		t.Fatalf("expected error response: %v", resp)
	} else if resp.Record != nil {
		t.Fatalf("unexpected record pointer: %v", resp.Record)
	} else if resp.Msg != "record 666 not found." {
		t.Fatalf("unexpected message: %s", resp.Msg)
	}
}

func TestServiceListRecordsSinglePage(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	list := pb.ListRequest{
		Page:    1,
		PerPage: testRecords,
	}

	if svc, err := NewClient(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.ListRecords(context.TODO(), &list); err != nil {
		t.Fatal(err)
	} else if resp.Total != testRecords {
		t.Fatalf("expected %d total records, got %d", testRecords, resp.Total)
	} else if resp.Pages != 1 {
		t.Fatalf("expected 1 page, got %d", resp.Pages)
	} else if len(resp.Records) != testRecords {
		t.Fatalf("expected %d total records, got %d", testRecords, len(resp.Records))
	} else {
		for _, r := range resp.Records {
			if testRecord.Id = r.Id; !proto.Equal(r, &testRecord) {
				t.Fatalf("unexpected record: %v", r)
			}
		}
	}
}

func TestServiceListRecordsMultiPage(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	list := pb.ListRequest{
		Page:    1,
		PerPage: 2,
	}

	if svc, err := NewClient(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.ListRecords(context.TODO(), &list); err != nil {
		t.Fatal(err)
	} else if resp.Total != testRecords {
		t.Fatalf("expected %d total records, got %d", testRecords, resp.Total)
	} else if resp.Pages != 3 {
		t.Fatalf("expected 3 pages got %d", resp.Pages)
	} else if len(resp.Records) != 2 {
		t.Fatalf("expected %d total records, got %d", 2, len(resp.Records))
	} else {
		for _, r := range resp.Records {
			if testRecord.Id = r.Id; !proto.Equal(r, &testRecord) {
				t.Fatalf("unexpected record: %v", r)
			}
		}
	}
}

func TestServiceListRecordsInvalidPage(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	list := pb.ListRequest{
		Page:    100000,
		PerPage: 2,
	}

	if svc, err := NewClient(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.ListRecords(context.TODO(), &list); err != nil {
		t.Fatal(err)
	} else if resp.Total != testRecords {
		t.Fatalf("expected %d total records, got %d", testRecords, resp.Total)
	} else if resp.Pages != 3 {
		t.Fatalf("expected 3 pages got %d", resp.Pages)
	} else if len(resp.Records) != 0 {
		t.Fatalf("expected %d records, got %d", 0, len(resp.Records))
	}
}

func TestServiceDeleteRecord(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	svc, err := NewClient(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < testRecords; i++ {
		id := uint64(i + 1)
		if resp, err := svc.DeleteRecord(context.TODO(), &pb.ById{Id: id}); err != nil {
			t.Fatal(err)
		} else if !resp.Success {
			t.Fatalf("expected success response: %v", resp)
		} else if resp.Record != nil {
			t.Fatalf("unexpected record pointer: %v", resp.Record)
		} else if network.orchestrators[0].svc.NumRecords() != testRecords-int(id) {
			t.Fatalf("inconsistent records storage size of %d", network.orchestrators[0].svc.NumRecords())
		}
	}

	if network.orchestrators[0].svc.NumRecords() != 0 {
		t.Fatalf("expected empty records storage, found %d instead", network.orchestrators[0].svc.NumRecords())
	}

	teardown(t)

	if _, err := NewClient(testFolder); err != nil {
		t.Fatal(err)
	} else if network.orchestrators[0].svc.NumRecords() != 0 {
		t.Fatalf("%d dat files left on disk", network.orchestrators[0].svc.NumRecords())
	}
}

func TestServiceDeleteRecordWithInvalidId(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := NewClient(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.DeleteRecord(context.TODO(), &pb.ById{Id: 666}); err != nil {
		t.Fatal(err)
	} else if resp.Success {
		t.Fatalf("expected error response: %v", resp)
	} else if resp.Record != nil {
		t.Fatalf("unexpected record pointer: %v", resp.Record)
	} else if resp.Msg != "record 666 not found." {
		t.Fatalf("unexpected message: %s", resp.Msg)
	}
}

func TestService_CreateRecordWithId(t *testing.T) {
	setupFolders(t)
	defer teardown(t)

	testRecord.Id = 5

	if svc, err := NewInternalClient(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.CreateRecordWithId(context.TODO(), &testRecord); err != nil {
		t.Fatal(err)
	} else if !resp.Success {
		t.Fatalf("expected success response: %v", resp)
	} else if resp.Record != nil {
		t.Fatalf("unexpected record pointer: %v", resp.Record)
	} else if resp.Msg != "5" {
		t.Fatalf("unexpected response message: %s", resp.Msg)
	}
}

func TestService_CreateRecordsWithId(t *testing.T) {
	setupFolders(t)
	defer teardown(t)

	arg := &pb.Records{Records: make([]*pb.Record, 0)}

	for i := 0; i < 5; i++ {
		rec := &pb.Record{Id: uint64(i + 1)}
		rec.Data = testRecord.Data
		rec.Meta = testRecord.Meta
		arg.Records = append(arg.Records, rec)
	}

	svc, err := NewInternalClient(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	if resp, err := svc.CreateRecordsWithId(context.TODO(), arg); err != nil {
		t.Fatal(err)
	} else if !resp.Success {
		t.Fatalf("expected success response: %v", resp)
	}
}

func TestService_DeleteRecords(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	svc, err := NewInternalClient(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	arg := &pb.RecordIds{Ids: make([]uint64, 0)}

	for i := 0; i < testRecords; i++ {
		arg.Ids = append(arg.Ids, uint64(i+1))
	}

	if resp, err := svc.DeleteRecords(context.TODO(), arg); err != nil {
		t.Fatal(err)
	} else if !resp.Success {
		t.Fatalf("expected success response: %v", resp)
	}
}
