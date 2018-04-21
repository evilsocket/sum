package storage

import (
	"testing"

	"github.com/golang/protobuf/proto"
)

func BenchmarkIndexLoad(b *testing.B) {
	setupRecords(b, true, false)
	defer teardownRecords(b)

	i := setupIndex(testFolder)
	for n := 0; n < b.N; n++ {
		if err := i.Load(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIndexPathForId(b *testing.B) {
	i := setupIndex("/foo")
	for n := 0; n < b.N; n++ {
		if s := i.pathForID(uint64(n%666) + 1); s == "" {
			b.Fatal("just to make sure the compiler doesn't optimize it away :D")
		}
	}
}

func BenchmarkIndexPathFor(b *testing.B) {
	i := setupIndex("/foo")
	for n := 0; n < b.N; n++ {
		testRecord.Id = uint64(n%666) + 1
		if s := i.pathFor(&testRecord); s == "" {
			b.Fatal("just to make sure the compiler doesn't optimize it away :D")
		}
	}
}

func BenchmarkIndexForEach(b *testing.B) {
	setupRecords(b, true, false)
	defer teardownRecords(b)

	i := setupIndex(testFolder)
	if err := i.Load(); err != nil {
		b.Fatal(err)
	}

	for n := 0; n < b.N; n++ {
		i.ForEach(func(m proto.Message) {})
	}
}

func BenchmarkIndexCreateRecord(b *testing.B) {
	setupRecords(b, false, false)
	defer teardownRecords(b)

	i := setupIndex(testFolder)
	if err := i.Load(); err != nil {
		b.Fatal(err)
	}

	for n := 0; n < b.N; n++ {
		if err := i.Create(&testRecord); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIndexUpdateRecord(b *testing.B) {
	setupRecords(b, true, false)
	defer teardownRecords(b)

	i := setupIndex(testFolder)
	if err := i.Load(); err != nil {
		b.Fatal(err)
	}

	for n := 0; n < b.N; n++ {
		updatedRecord.Id = uint64(n%testRecords) + 1
		if err := i.Update(&updatedRecord); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIndexFindRecord(b *testing.B) {
	setupRecords(b, true, false)
	defer teardownRecords(b)

	i := setupIndex(testFolder)
	if err := i.Load(); err != nil {
		b.Fatal(err)
	}

	for n := 0; n < b.N; n++ {
		testRecord.Id = uint64(n%666) + 1
		_ = i.Find(testRecord.Id)
	}
}

func BenchmarkIndexDeleteRecord(b *testing.B) {
	defer teardownRecords(b)

	var idx *Index
	for i := 0; i < b.N; i++ {
		// this is not entirely ok as once every 5 times
		// we neeed to recreate and reload records, which
		// increases the operations being benchmarked
		id := uint64(i%testRecords) + 1
		if id == 1 {
			setupRecords(b, true, false)
			idx = setupIndex(testFolder)
			if err := idx.Load(); err != nil {
				b.Fatal(err)
			}
		}

		if m := idx.Delete(id); m == nil {
			b.Fatal("unexpected nil response")
		}
	}
}
