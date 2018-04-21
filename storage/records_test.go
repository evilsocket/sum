package storage

import (
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"

	pb "github.com/evilsocket/sum/proto"

	"github.com/golang/protobuf/proto"
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
	log.SetOutput(ioutil.Discard)

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

	records.ForEach(func(m proto.Message) {
		r := m.(*pb.Record)
		// id was updated while saving the record
		if r.Id = testRecord.Id; !reflect.DeepEqual(*r, testRecord) {
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

	updatedRecord.Id = 4
	if err := records.Update(&updatedRecord); err != nil {
		t.Fatal(err)
	} else if record := records.Find(updatedRecord.Id); record == nil {
		t.Fatalf("expected record with id %d", updatedRecord.Id)
	} else if !reflect.DeepEqual(*record, updatedRecord) {
		t.Fatal("records should match")
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

func TestRecordsDeleteWithInvalidId(t *testing.T) {
	setupRecords(t, false, false)
	defer teardownRecords(t)

	records, err := LoadRecords(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < testRecords; i++ {
		if deleted := records.Delete(uint64(i + 1)); deleted != nil {
			t.Fatalf("record with id %d was not expected to be found", i+1)
		}
	}
}
