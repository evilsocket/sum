package service

import (
	"context"
	"testing"

	pb "github.com/evilsocket/sum/proto"
)

func BenchmarkServiceCreateRecord(b *testing.B) {
	setupFolders(b)
	defer teardown(b)

	svc, err := New(testFolder, "", "")
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.TODO()
	for i := 0; i < b.N; i++ {
		if _, err := svc.CreateRecord(ctx, &testRecord); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkServiceUpdateRecord(b *testing.B) {
	setup(b, true, true)
	defer teardown(b)

	svc, err := New(testFolder, "", "")
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.TODO()
	for i := 0; i < b.N; i++ {
		updatedRecord.Id = uint64(i%testRecords) + 1
		if _, err := svc.UpdateRecord(ctx, &updatedRecord); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkServiceReadRecord(b *testing.B) {
	setup(b, true, true)
	defer teardown(b)

	svc, err := New(testFolder, "", "")
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.TODO()
	for i := 0; i < b.N; i++ {
		byID.Id = uint64(i%testRecords) + 1
		if _, err := svc.ReadRecord(ctx, &byID); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkServiceListRecords(b *testing.B) {
	setup(b, true, true)
	defer teardown(b)

	svc, err := New(testFolder, "", "")
	if err != nil {
		b.Fatal(err)
	}

	list := pb.ListRequest{
		Page:    1,
		PerPage: testRecords,
	}

	ctx := context.TODO()
	for i := 0; i < b.N; i++ {
		if _, err := svc.ListRecords(ctx, &list); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkServiceDeleteRecord(b *testing.B) {
	defer teardown(b)

	var svc *Service
	var err error

	ctx := context.TODO()
	for i := 0; i < b.N; i++ {
		id := uint64(i%testRecords) + 1
		if id == 1 {
			setup(b, true, true)
			if svc, err = New(testFolder, "", ""); err != nil {
				b.Fatal(err)
			}
		}

		if _, err := svc.DeleteRecord(ctx, &pb.ById{Id: id}); err != nil {
			b.Fatal(err)
		}
	}
}
