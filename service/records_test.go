package service

import (
	"context"
	"reflect"
	"testing"

	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/storage"
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

func TestErrRecordResponse(t *testing.T) {
	if r := errRecordResponse("test %d", 123); r.Success {
		t.Fatal("success should be false")
	} else if r.Msg != "test 123" {
		t.Fatalf("unexpected message: %s", r.Msg)
	} else if r.Record != nil {
		t.Fatalf("unexpected record pointer: %v", r.Record)
	}
}

func TestCreateRecord(t *testing.T) {
	setupFolders(t)
	defer teardown(t)

	if svc, err := New(testFolder); err != nil {
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

func BenchmarkCreateRecord(b *testing.B) {
	setupFolders(b)
	defer teardown(b)

	svc, err := New(testFolder)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		if resp, err := svc.CreateRecord(context.TODO(), &testRecord); err != nil {
			b.Fatal(err)
		} else if !resp.Success {
			b.Fatalf("expected success response: %v", resp)
		}
	}
}

func TestCreateRecordNotUniqueId(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	svc, err := New(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	// ok this is kinda cheating, but i want full coverage
	svc.records.NextID(1)
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

func TestUpdateRecord(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	svc, err := New(testFolder)
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
	} else if stored := svc.records.Find(updatedRecord.Id); stored == nil {
		t.Fatal("expected stored record with id 1")
	} else if !reflect.DeepEqual(*stored, updatedRecord) {
		t.Fatal("record has not been updated as expected")
	}
}

func BenchmarkUpdateRecord(b *testing.B) {
	setup(b, true, true)
	defer teardown(b)

	svc, err := New(testFolder)
	if err != nil {
		b.Fatal(err)
	}

	updatedRecord.Id = 1
	for i := 0; i < b.N; i++ {
		if resp, err := svc.UpdateRecord(context.TODO(), &updatedRecord); err != nil {
			b.Fatal(err)
		} else if !resp.Success {
			b.Fatalf("expected success response: %v", resp)
		}
	}
}

func TestUpdateRecordWithInvalidId(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	svc, err := New(testFolder)
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

func TestReadRecord(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.ReadRecord(context.TODO(), &byID); err != nil {
		t.Fatal(err)
	} else if !resp.Success {
		t.Fatalf("expected success response: %v", resp)
	} else if resp.Record == nil {
		t.Fatal("expected record pointer")
	} else if testRecord.Id = byID.Id; !reflect.DeepEqual(*resp.Record, testRecord) {
		t.Fatalf("unexpected record: %v", resp.Record)
	}
}

func BenchmarkReadRecord(b *testing.B) {
	setup(b, true, true)
	defer teardown(b)

	svc, err := New(testFolder)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		if resp, err := svc.ReadRecord(context.TODO(), &byID); err != nil {
			b.Fatal(err)
		} else if !resp.Success {
			b.Fatalf("expected success response: %v", resp)
		}
	}
}

func TestReadRecordWithInvalidId(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder); err != nil {
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

func TestDeleteRecord(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	svc, err := New(testFolder)
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
		} else if svc.NumRecords() != uint64(testRecords)-id {
			t.Fatalf("inconsistent records storage size of %d", svc.NumRecords())
		}
	}

	if svc.NumRecords() != 0 {
		t.Fatalf("expected empty records storage, found %d instead", svc.NumRecords())
	} else if doublecheck, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if doublecheck.NumRecords() != 0 {
		t.Fatalf("%d dat files left on disk", doublecheck.NumRecords())
	}
}

func BenchmarkDeleteRecord(b *testing.B) {
	var svc *Service
	var err error

	defer teardown(b)
	for i := 0; i < b.N; i++ {
		// this is not entirely ok as once every 5 times
		// we neeed to recreate and reload records, which
		// increases the operations being benchmarked
		id := uint64(i%testRecords) + 1
		if id == 1 {
			setup(b, true, true)
			if svc, err = New(testFolder); err != nil {
				b.Fatal(err)
			}
		}

		if resp, err := svc.DeleteRecord(context.TODO(), &pb.ById{Id: id}); err != nil {
			b.Fatal(err)
		} else if !resp.Success {
			b.Fatalf("expected success response: %v", resp)
		}
	}
}

func TestDeleteRecordWithInvalidId(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder); err != nil {
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
