package storage

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	pb "github.com/evilsocket/sum/proto"
)

var (
	testCorruptedRecord = testFolder + "/666.dat"
	updatedRecord       = pb.Record{
		Id:   555,
		Data: []float32{0.5, 0.5, 0.5},
		Meta: map[string]string{"555": "555"},
	}
)

func setupRecords(t testing.TB, withValid bool, withCorrupted bool) {
	// start clean
	teardownRecords(t)

	if err := os.MkdirAll(testFolder, 0755); err != nil {
		t.Fatalf("Error creating %s: %s", testFolder, err)
	}

	dummy, err := LoadRecords(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	if withValid {
		for i := 1; i <= testRecords; i++ {
			if err := dummy.Create(&testRecord); err != nil {
				t.Fatalf("Error creating record: %s", err)
			}
		}
	}

	if withCorrupted {
		if err := ioutil.WriteFile(testCorruptedRecord, []byte("i'm corrupted inside"), 0755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoadRecords(t *testing.T) {
	setupRecords(t, true, false)
	defer teardownRecords(t)

	records, err := LoadRecords(testFolder)
	if err != nil {
		t.Fatal(err)
	} else if records == nil {
		t.Fatal("expected valid records storage")
	} else if records.Size() != testRecords {
		t.Fatalf("expected %d records, %d found in %s", testRecords, records.Size(), testFolder)
	}

	records.ForEach(func(r *pb.Record) {
		// id was updated while saving the record
		if r.Id = testRecord.Id; reflect.DeepEqual(*r, testRecord) == false {
			t.Fatalf("records should be the same here")
		}
	})
}

func TestLoadRecordsWithCorruptedData(t *testing.T) {
	setupRecords(t, false, true)
	defer teardownRecords(t)

	if records, err := LoadRecords("/lulzlulz"); err == nil {
		t.Fatal("expected error")
	} else if records != nil {
		t.Fatal("expected no storage loaded")
	} else if records, err := LoadRecords(testFolder); err == nil {
		t.Fatal("expected error due to broken record dat file")
	} else if records != nil {
		t.Fatal("expected no storage loaded due to corrupted record dat file")
	}
}

func TestRecordsCreate(t *testing.T) {
	setupRecords(t, false, false)
	defer teardownRecords(t)

	records, err := LoadRecords(testFolder)
	if err != nil {
		t.Fatal(err)
	} else if records.Size() != 0 {
		t.Fatal("expected empty record storage")
	} else if err := records.Create(&testRecord); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkRecordsCreate(b *testing.B) {
	setupRecords(b, false, false)
	defer teardownRecords(b)

	records, err := LoadRecords(testFolder)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		if err := records.Create(&testRecord); err != nil {
			b.Fatal(err)
		}
	}
}

func TestRecordsCreateNotUniqueId(t *testing.T) {
	setupRecords(t, true, false)
	defer teardownRecords(t)

	records, err := LoadRecords(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	// ok this is kinda cheating, but i want full coverage
	records.nextId = uint64(1)
	if err := records.Create(&testRecord); err == nil {
		t.Fatalf("expected error for non unique record id")
	}
}

func TestRecordsFind(t *testing.T) {
	setupRecords(t, true, false)
	defer teardownRecords(t)

	records, err := LoadRecords(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < testRecords; i++ {
		if record := records.Find(uint64(i + 1)); record == nil {
			t.Fatalf("record with id %d not found", i)
		}
	}
}

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
			b.Fatalf("record with id %d not found", i)
		}
	}
}

func TestRecordsFindWithInvalidId(t *testing.T) {
	setupRecords(t, false, false)
	defer teardownRecords(t)

	records, err := LoadRecords(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < testRecords; i++ {
		if record := records.Find(uint64(i + 1)); record != nil {
			t.Fatalf("record with id %d was not expected to be found", i)
		}
	}
}

func TestRecordsUpdate(t *testing.T) {
	setupRecords(t, true, false)
	defer teardownRecords(t)

	records, err := LoadRecords(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	updatedRecord.Id = 1
	if err := records.Update(&updatedRecord); err != nil {
		t.Fatal(err)
	}

	if stored := records.Find(updatedRecord.Id); stored == nil {
		t.Fatal("expected stored record with id 1")
	} else if reflect.DeepEqual(*stored, updatedRecord) == false {
		t.Fatal("record has not been updated as expected")
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

func TestRecordsUpdateInvalidId(t *testing.T) {
	setupRecords(t, true, false)
	defer teardownRecords(t)

	records, err := LoadRecords(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	updatedRecord.Id = ^uint64(0)
	if err := records.Update(&updatedRecord); err == nil {
		t.Fatal("expected error due to invalid id")
	}
}

func TestRecordsDelete(t *testing.T) {
	setupRecords(t, true, false)
	defer teardownRecords(t)

	records, err := LoadRecords(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < testRecords; i++ {
		id := uint64(i + 1)
		if deleted := records.Delete(id); deleted == nil {
			t.Fatalf("record with id %d not found", id)
		} else if deleted.Id != id {
			t.Fatalf("should have deleted record with id %d, id is %d instead", id, deleted.Id)
		} else if records.Size() != uint64(testRecords)-id {
			t.Fatalf("inconsistent records storage size of %d", records.Size())
		} else if _, err := os.Stat(records.pathFor(deleted)); err == nil {
			t.Fatalf("record %d data file was not deleted", deleted.Id)
		}
	}

	if records.Size() != 0 {
		t.Fatalf("expected empty records storage, found %d instead", records.Size())
	} else if doublecheck, err := LoadRecords(testFolder); err != nil {
		t.Fatal(err)
	} else if doublecheck.Size() != 0 {
		t.Fatalf("%d dat files left on disk", doublecheck.Size())
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

func TestRecordsDeleteWithInvalidId(t *testing.T) {
	setupRecords(t, false, false)
	defer teardownRecords(t)

	records, err := LoadRecords(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < testRecords; i++ {
		if deleted := records.Delete(uint64(i + 1)); deleted != nil {
			t.Fatalf("record with id %d was not expected to be found", i)
		}
	}
}
