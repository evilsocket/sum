package storage

import (
	"testing"
)

func BenchmarkRecordsFind(b *testing.B) {
	setupRecords(b, true, false)
	defer teardownRecords(b)

	records, err := LoadRecords(testFolder)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		id := uint64(i%testRecords) + 1
		if record := records.Find(id); record == nil {
			b.Fatalf("record with id %d not found", id)
		}
	}
}

func BenchmarkRecordsUpdate(b *testing.B) {
	setupRecords(b, true, false)
	defer teardownRecords(b)

	records, err := LoadRecords(testFolder)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		updatedRecord.Id = uint64(i%testRecords) + 1
		if err := records.Update(&updatedRecord); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRecordsDelete(b *testing.B) {
	defer teardownRecords(b)

	var records *Records
	var err error

	for i := 0; i < b.N; i++ {
		// this is not entirely ok as once every 5 times
		// we neeed to recreate and reload records, which
		// increases the operations being benchmarked
		id := uint64(i%testRecords) + 1
		if id == 1 {
			setupRecords(b, true, false)
			if records, err = LoadRecords(testFolder); err != nil {
				b.Fatal(err)
			}
		}

		if deleted := records.Delete(id); deleted == nil {
			b.Fatalf("record with id %d not found", id)
		}
	}
}
