package main

import (
	"context"
	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/storage"
	"github.com/golang/protobuf/proto"
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
		t.Fatalf("expected %d total records, got %d", 0, len(resp.Records))
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
		} else if len(network.orchestrators[0].svc.recId2node) != testRecords-int(id) {
			t.Fatalf("inconsistent records storage size of %d", len(network.orchestrators[0].svc.recId2node))
		}
	}

	if len(network.orchestrators[0].svc.recId2node) != 0 {
		t.Fatalf("expected empty records storage, found %d instead", len(network.orchestrators[0].svc.recId2node))
	}

	teardown(t)

	if _, err := NewClient(testFolder); err != nil {
		t.Fatal(err)
	} else if len(network.orchestrators[0].svc.recId2node) != 0 {
		t.Fatalf("%d dat files left on disk", len(network.orchestrators[0].svc.recId2node))
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
