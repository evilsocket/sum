package service

import (
	"context"
	"testing"

	pb "github.com/evilsocket/sum/proto"
)

func BenchmarkServiceCreateOracle(b *testing.B) {
	setupFolders(b)
	defer teardown(b)

	svc, err := New(testFolder, "", "")
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.TODO()
	for i := 0; i < b.N; i++ {
		if _, err := svc.CreateOracle(ctx, &testOracle); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkServiceUpdateOracle(b *testing.B) {
	setup(b, true, true)
	defer teardown(b)

	svc, err := New(testFolder, "", "")
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.TODO()
	for i := 0; i < b.N; i++ {
		updatedOracle.Id = uint64(i%testOracles) + 1
		if _, err := svc.UpdateOracle(ctx, &updatedOracle); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkServiceReadOracle(b *testing.B) {
	setup(b, true, true)
	defer teardown(b)

	svc, err := New(testFolder, "", "")
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.TODO()
	for i := 0; i < b.N; i++ {
		byID.Id = uint64(i%testOracles) + 1
		if _, err := svc.ReadOracle(ctx, &byID); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkServiceFindOracle(b *testing.B) {
	setup(b, true, true)
	defer teardown(b)

	svc, err := New(testFolder, "", "")
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.TODO()
	for i := 0; i < b.N; i++ {
		if _, err := svc.FindOracle(ctx, &byName); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkServiceDeleteOracle(b *testing.B) {
	var svc *Service
	var err error

	defer teardown(b)
	ctx := context.TODO()
	for i := 0; i < b.N; i++ {
		// this is not entirely ok as once every 5 times
		// we neeed to recreate and reload oracles, which
		// increases the operations being benchmarked
		id := uint64(i%testOracles) + 1
		if id == 1 {
			setup(b, true, true)
			if svc, err = New(testFolder, "", ""); err != nil {
				b.Fatal(err)
			}
		}

		if _, err := svc.DeleteOracle(ctx, &pb.ById{Id: id}); err != nil {
			b.Fatal(err)
		}
	}
}
