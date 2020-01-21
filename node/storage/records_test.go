package storage

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"log"
	"os"
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

	records.ForEach(func(m proto.Message) error {
		r := m.(*pb.Record)
		// id was updated while saving the record
		if r.Id = testRecord.Id; !sameRecord(*r, testRecord) {
			t.Fatalf("records should be the same here")
		}
		return nil
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
	} else if !sameRecord(*record, updatedRecord) {
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
		} else if records.Size() != testRecords-int(id) {
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

func TestRecords_CreateMany(t *testing.T) {
	setupRecords(t, false, false)
	defer teardownRecords(t)

	records, err := LoadRecords(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("ok", func(t *testing.T) {
		err := records.CreateMany(&pb.Records{Records: []*pb.Record{&testRecord}})
		require.NoError(t, err)
		defer records.Delete(testRecord.Id)
		require.Equal(t, records.Size(), 1)
	})

	t.Run("no meta", func(t *testing.T) {
		rec := &pb.Record{Data: []float32{0, 1, 0, 2}}

		err := records.CreateMany(&pb.Records{Records: []*pb.Record{rec}})
		require.NoError(t, err)
		defer records.Delete(rec.Id)
		require.Equal(t, records.Size(), 1)
	})

	t.Run("empty", func(t *testing.T) {
		err := records.CreateMany(&pb.Records{})
		require.NoError(t, err)
		require.Zero(t, records.Size())
	})

	t.Run("ko", func(t *testing.T) {
		defer func(oldDataPath string) {
			records.dataPath = oldDataPath
		}(records.dataPath)
		records.dataPath = "/does/not/exist"

		err := records.CreateMany(&pb.Records{Records: []*pb.Record{&testRecord}})
		require.Error(t, err)
		require.Zero(t, records.Size())
	})

	t.Run("WithId", func(t *testing.T) {
		testRecord.Id = 5
		err := records.CreateManyWIthId([]*pb.Record{&testRecord})
		require.NoError(t, err)
		defer records.Delete(5)
		require.Equal(t, records.Size(), 1)
		require.Contains(t, records.index, uint64(5))
		require.True(t, sameRecord(testRecord, *((records.index[5]).(*pb.Record))),
			"expected record to be '%v', got '%v'", testRecord, records.index[5])
	})
}

func TestRecords_FindBy(t *testing.T) {
	setupRecords(t, true, false)
	defer teardownRecords(t)

	records, err := LoadRecords(testFolder)
	require.NoError(t, err)

	t.Run("matches", func(t *testing.T) {
		matches := records.FindBy("666", "666")
		require.Len(t, matches, testRecords)
	})

	t.Run("no match", func(t *testing.T) {
		matches := records.FindBy("nope", "nada")
		require.Empty(t, matches)
	})

	t.Run("key found, no match", func(t *testing.T) {
		matches := records.FindBy("666", "nada")
		require.Empty(t, matches)
	})
}

func TestRecords_DeleteMany(t *testing.T) {
	setupRecords(t, true, false)
	defer teardownRecords(t)

	records, err := LoadRecords(testFolder)
	require.NoError(t, err)

	allIds := make([]uint64, 0, len(records.index))
	for id := range records.index {
		allIds = append(allIds, id)
	}

	deleted := records.DeleteMany(allIds)
	require.Len(t, deleted, testRecords)
	require.Zero(t, records.Size())
}
