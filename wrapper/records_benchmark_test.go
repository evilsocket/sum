package wrapper

import (
	"testing"

	"github.com/evilsocket/sum/storage"
)

func BenchmarkWrappedRecordsFind(b *testing.B) {
	setupRecords(b, true)
	defer teardownRecords(b)

	records, err := storage.LoadRecords(testFolder)
	if err != nil {
		b.Fatal(err)
	}

	wrapped := WrapRecords(records)
	for i := 0; i < b.N; i++ {
		id := uint64(i%testRecords) + 1
		if r := wrapped.Find(id); r.IsNull() {
			b.Fatalf("wrapped record with id %d not found", id)
		}
	}
}

func BenchmarkWrappedRecordsAll(b *testing.B) {
	setupRecords(b, true)
	defer teardownRecords(b)

	records, err := storage.LoadRecords(testFolder)
	if err != nil {
		b.Fatal(err)
	}
	wrapped := WrapRecords(records)
	for i := 0; i < b.N; i++ {
		if all := wrapped.All(); len(all) != testRecords {
			b.Fatalf("expected %d wrapped records, got %d", testRecords, len(all))
		}
	}
}

func BenchmarkWrappedRecordsAllBut(b *testing.B) {
	setupRecords(b, true)
	defer teardownRecords(b)

	records, err := storage.LoadRecords(testFolder)
	if err != nil {
		b.Fatal(err)
	}

	reference := records.Find(1)
	if reference == nil {
		b.Fatal("expected record with id 1")
	}

	wrapped := WrapRecords(records)
	wRef := WrapRecord(reference)

	for i := 0; i < b.N; i++ {
		if allBut := wrapped.AllBut(wRef); len(allBut) != testRecords-1 {
			b.Fatalf("expected %d wrapped records, got %d", testRecords-1, len(allBut))
		}
	}
}
