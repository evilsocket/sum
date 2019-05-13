package storage

import (
	"errors"
	"math/rand"
	"os"
	"reflect"
	"testing"

	pb "github.com/evilsocket/sum/proto"

	"github.com/golang/protobuf/proto"
)

var (
	errFound = errors.New("record found")
	errTest  = errors.New("test error")
)

func setupIndex(folder string) *Index {
	return WithDriver(folder, RecordDriver{})
}

func TestNewIndexWithRecordDriver(t *testing.T) {
	if i := setupIndex("12345"); i.Size() != 0 {
		t.Fatalf("unexpected index size %d", i.Size())
	}
}

func TestIndexLoad(t *testing.T) {
	setupRecords(t, true, false)
	defer teardownRecords(t)

	i := setupIndex(testFolder)
	if err := i.Load(); err != nil {
		t.Fatal(err)
	} else if i.Size() != testRecords {
		t.Fatalf("expected %d records, got %d", testRecords, i.Size())
	}

	for id := uint64(1); id <= testRecords; id++ {
		testRecord.Id = id
		if m := i.Find(id); m == nil {
			t.Fatalf("expected record %d not found", id)
		} else if o := m.(*pb.Record); !sameRecord(*o, testRecord) {
			t.Fatalf("records do not match:\nexpected %v\ngot %v", testRecord, *o)
		}
	}
}

func TestIndexLoadWithNoFolder(t *testing.T) {
	i := setupIndex("/ilulzsomuch")
	if err := i.Load(); err == nil {
		t.Fatal("expected error")
	} else if i.Size() != 0 {
		t.Fatalf("unexpected index size %d", i.Size())
	}

	i = setupIndex("/dev/random")
	if err := i.Load(); err == nil {
		t.Fatal("expected error")
	} else if i.Size() != 0 {
		t.Fatalf("unexpected index size %d", i.Size())
	}
}

func TestIndexPathForId(t *testing.T) {
	i := setupIndex("/foo")
	if path := i.pathForID(1234); path != "/foo/1234.dat" {
		t.Fatalf("unpexpected path: %s", path)
	}
}

func TestIndexPathFor(t *testing.T) {
	i := setupIndex("/foo")
	testRecord.Id = 1234
	if path := i.pathFor(&testRecord); path != "/foo/1234.dat" {
		t.Fatalf("unpexpected path: %s", path)
	}
}

func TestIndexForEach(t *testing.T) {
	setupRecords(t, true, false)
	defer teardownRecords(t)

	i := setupIndex(testFolder)
	if err := i.Load(); err != nil {
		t.Fatal(err)
	} else if i.Size() != testRecords {
		t.Fatalf("expected %d records, got %d", testRecords, i.Size())
	}

	i.ForEach(func(m proto.Message) error {
		record := m.(*pb.Record)
		testRecord.Id = record.Id
		if !sameRecord(*record, testRecord) {
			t.Fatal("records should match")
		}
		return nil
	})
}

func TestIndexForEachShouldStopLoop(t *testing.T) {
	setupRecords(t, true, false)
	defer teardownRecords(t)

	i := setupIndex(testFolder)
	if err := i.Load(); err != nil {
		t.Fatal(err)
	} else if i.Size() != testRecords {
		t.Fatalf("expected %d records, got %d", testRecords, i.Size())
	}

	if err := i.ForEach(func(m proto.Message) error { return errTest }); err != errTest {
		t.Fatalf("expected %v, got %v", errTest, err)
	}
}

func TestIndexObjects(t *testing.T) {
	setupRecords(t, true, false)
	defer teardownRecords(t)

	i := setupIndex(testFolder)
	if err := i.Load(); err != nil {
		t.Fatal(err)
	}

	asSlice := i.Objects()
	inSlice := len(asSlice)
	if inSlice != testRecords {
		t.Fatalf("expected %d records, got %d", testRecords, inSlice)
	} else if inSlice != i.Size() {
		t.Fatalf("expected %d records, got %d", i.Size(), inSlice)
	}

	for _, objInSlice := range asSlice {
		err := i.ForEach(func(m proto.Message) error {
			if reflect.DeepEqual(m, objInSlice) {
				return errFound
			}
			return nil
		})
		if err != errFound {
			t.Fatal("object in slice not found in index")
		}
	}
}

func TestIndexCreateRecord(t *testing.T) {
	setupRecords(t, false, false)
	defer teardownRecords(t)

	i := setupIndex(testFolder)
	if err := i.Load(); err != nil {
		t.Fatal(err)
	} else if i.Size() != 0 {
		t.Fatalf("expected %d records, got %d", 0, i.Size())
	} else if err := i.Create(&testRecord); err != nil {
		t.Fatal(err)
	} else if i.Size() != 1 {
		t.Fatalf("expected %d records, got %d", 1, i.Size())
	} else if m := i.Find(testRecord.Id); m == nil {
		t.Fatalf("expected record with id %d", testRecord.Id)
	} else if r := m.(*pb.Record); !sameRecord(*r, testRecord) {
		t.Fatal("records should match")
	}
}

func TestIndexCreateRecordWithId(t *testing.T) {
	setupRecords(t, false, false)
	defer teardownRecords(t)

	i := setupIndex(testFolder)
	if err := i.Load(); err != nil {
		t.Fatal(err)
	} else if i.Size() != 0 {
		t.Fatalf("expected %d records, got %d", 0, i.Size())
	}

	for {
		testRecord.Id = rand.Uint64()
		if _, found := i.index[testRecord.Id]; !found {
			break
		}
	}

	if err := i.CreateWithId(&testRecord); err != nil {
		t.Fatal(err)
	} else if i.Size() != 1 {
		t.Fatalf("expected %d records, got %d", 1, i.Size())
	} else if m := i.Find(testRecord.Id); m == nil {
		t.Fatalf("expected record with id %d", testRecord.Id)
	} else if r := m.(*pb.Record); !sameRecord(*r, testRecord) {
		t.Fatal("records should match")
	}
}

func TestIndexCreateRecordWithId_InvalidId(t *testing.T) {
	setupRecords(t, true, false)
	defer teardownRecords(t)

	i := setupIndex(testFolder)
	if err := i.Load(); err != nil {
		t.Fatal(err)
	} else if i.Size() != testRecords {
		t.Fatalf("expected %d records, got %d", testRecords, i.Size())
	}
	testRecord.Id = testRecords

	if err := i.CreateWithId(&testRecord); err != ErrInvalidID {
		t.Fatalf("expected invalid id error, got %v", err)
	}
}

func TestIndexCreateRecordWithoutFolder(t *testing.T) {
	i := setupIndex("/ilulzsomuch")
	if err := i.Load(); err == nil {
		t.Fatal("expected error")
	} else if i.Size() != 0 {
		t.Fatalf("unexpected index size %d", i.Size())
	} else if err := i.Create(&testRecord); err == nil {
		t.Fatalf("expected error")
	} else if i.Size() != 0 {
		t.Fatalf("unexpected index size %d", i.Size())
	}
}

func TestIndexCreateRecordWithInvalidId(t *testing.T) {
	setupRecords(t, true, false)
	defer teardownRecords(t)

	i := setupIndex(testFolder)
	if err := i.Load(); err != nil {
		t.Fatal(err)
	} else if i.Size() != testRecords {
		t.Fatalf("expected %d records, got %d", testRecords, i.Size())
	}

	i.NextID(1)
	if err := i.Create(&testRecord); err != ErrInvalidID {
		t.Fatalf("expected invalid id error, got %v", err)
	}
}

func TestIndexUpdateRecord(t *testing.T) {
	setupRecords(t, true, false)
	defer teardownRecords(t)

	i := setupIndex(testFolder)
	if err := i.Load(); err != nil {
		t.Fatal(err)
	} else if i.Size() != testRecords {
		t.Fatalf("expected %d records, got %d", testRecords, i.Size())
	}

	updatedRecord.Id = 4
	if err := i.Update(&updatedRecord); err != nil {
		t.Fatal(err)
	} else if i.Size() != testRecords {
		t.Fatalf("expected %d records, got %d", testRecords, i.Size())
	} else if m := i.Find(updatedRecord.Id); m == nil {
		t.Fatalf("expected record with id %d", updatedRecord.Id)
	} else if r := m.(*pb.Record); !sameRecord(*r, updatedRecord) {
		t.Fatal("records should match")
	}
}

func TestIndexUpdateRecordWithInvalidId(t *testing.T) {
	setupRecords(t, false, false)
	defer teardownRecords(t)

	i := setupIndex(testFolder)
	if err := i.Load(); err != nil {
		t.Fatal(err)
	} else if i.Size() != 0 {
		t.Fatalf("expected %d records, got %d", 0, i.Size())
	}

	updatedRecord.Id = 666
	if err := i.Update(&updatedRecord); err != ErrRecordNotFound {
		t.Fatalf("expected record not found error, got %v", err)
	}
}

type faulty struct {
	RecordDriver
}

func (d faulty) Copy(mdst proto.Message, msrc proto.Message) error {
	return errTest
}

func TestIndexUpdateRecordWithCopyError(t *testing.T) {
	setupRecords(t, true, false)
	defer teardownRecords(t)

	updatedRecord.Id = 1
	i := WithDriver(testFolder, faulty{})
	if err := i.Load(); err != nil {
		t.Fatal(err)
	} else if i.Size() != testRecords {
		t.Fatalf("expected %d records, got %d", testRecords, i.Size())
	} else if err := i.Update(&updatedRecord); err != errTest {
		t.Fatalf("expected the test error, got %v", err)
	}
}

func TestIndexFindRecord(t *testing.T) {
	setupRecords(t, true, false)
	defer teardownRecords(t)

	testRecord.Id = 4
	i := setupIndex(testFolder)
	if err := i.Load(); err != nil {
		t.Fatal(err)
	} else if m := i.Find(testRecord.Id); m == nil {
		t.Fatalf("expected record with id %d", testRecord.Id)
	} else if r := m.(*pb.Record); !sameRecord(*r, testRecord) {
		t.Fatal("records should match")
	}
}

func TestIndexFindRecordWithInvalidId(t *testing.T) {
	setupRecords(t, true, false)
	defer teardownRecords(t)

	i := setupIndex(testFolder)
	if err := i.Load(); err != nil {
		t.Fatal(err)
	} else if m := i.Find(666); m != nil {
		t.Fatalf("expected nil, got %v", m)
	}
}

func TestIndexDeleteRecord(t *testing.T) {
	setupRecords(t, true, false)
	defer teardownRecords(t)

	i := setupIndex(testFolder)
	if err := i.Load(); err != nil {
		t.Fatal(err)
	}

	for n := 0; n < testRecords; n++ {
		id := uint64(n + 1)
		if m := i.Delete(id); m == nil {
			t.Fatalf("record with id %d not found", id)
		} else if deleted := m.(*pb.Record); deleted.Id != id {
			t.Fatalf("should have deleted record with id %d, id is %d instead", id, deleted.Id)
		} else if i.Size() != testRecords-int(id) {
			t.Fatalf("inconsistent index size of %d", i.Size())
		} else if _, err := os.Stat(i.pathFor(deleted)); err == nil {
			t.Fatalf("record %d data file was not deleted", deleted.Id)
		}
	}

	if i.Size() != 0 {
		t.Fatalf("expected empty index, found %d elements instead", i.Size())
	}
}

func TestIndexDeleteRecordWithInvalidId(t *testing.T) {
	setupRecords(t, false, false)
	defer teardownRecords(t)

	i := setupIndex(testFolder)
	if err := i.Load(); err != nil {
		t.Fatal(err)
	}

	for n := 0; n < testRecords; n++ {
		if m := i.Delete(uint64(n + 1)); m != nil {
			t.Fatalf("record with id %d was not expected to be found", n+1)
		}
	}
}
