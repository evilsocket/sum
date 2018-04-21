package storage

import (
	"reflect"
	"testing"

	pb "github.com/evilsocket/sum/proto"

	"github.com/golang/protobuf/proto"
)

func setupIndex(folder string) *Index {
	i := NewIndex(folder)
	i.Maker(func() proto.Message { return new(pb.Record) })
	i.Hasher(func(m proto.Message) uint64 { return m.(*pb.Record).Id })
	i.Marker(func(m proto.Message, mark uint64) { m.(*pb.Record).Id = mark })
	i.Copier(func(mold proto.Message, mnew proto.Message) error {
		old := mold.(*pb.Record)
		new := mnew.(*pb.Record)
		if new.Meta != nil {
			old.Meta = new.Meta
		}
		if new.Data != nil {
			old.Data = new.Data
		}
		return nil
	})
	return i
}

func TestNewIndex(t *testing.T) {
	if i := NewIndex("12345"); i.Size() != 0 {
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
		} else if o := m.(*pb.Record); reflect.DeepEqual(*o, testRecord) == false {
			t.Fatalf("records do not match:\nexpected %v\ngot %v", testRecord, *o)
		}
	}
}

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
	if path := i.pathForId(1234); path != "/foo/1234.dat" {
		t.Fatalf("unpexpected path: %s", path)
	}
}

func BenchmarkIndexPathForId(b *testing.B) {
	i := setupIndex("/foo")
	for n := 0; n < b.N; n++ {
		_ = i.pathForId(uint64(n%666) + 1)
	}
}

func TestIndexPathFor(t *testing.T) {
	i := setupIndex("/foo")
	testRecord.Id = 1234
	if path := i.pathFor(&testRecord); path != "/foo/1234.dat" {
		t.Fatalf("unpexpected path: %s", path)
	}
}

func BenchmarkIndexPathFor(b *testing.B) {
	i := setupIndex("/foo")
	for n := 0; n < b.N; n++ {
		testRecord.Id = uint64(n%666) + 1
		_ = i.pathFor(&testRecord)
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

	i.ForEach(func(m proto.Message) {
		record := m.(*pb.Record)
		testRecord.Id = record.Id
		if reflect.DeepEqual(*record, testRecord) == false {
			t.Fatal("records should match")
		}
	})
}

func BenchmarkIndexForEach(b *testing.B) {
	setupRecords(b, true, false)
	defer teardownRecords(b)

	i := setupIndex(testFolder)
	if err := i.Load(); err != nil {
		b.Fatal(err)
	} else if i.Size() != testRecords {
		b.Fatalf("expected %d records, got %d", testRecords, i.Size())
	}

	for n := 0; n < b.N; n++ {
		i.ForEach(func(m proto.Message) {})
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
	} else if r := m.(*pb.Record); reflect.DeepEqual(*r, testRecord) == false {
		t.Fatal("records should match")
	}
}

func BenchmarkIndexCreateRecord(b *testing.B) {
	setupRecords(b, false, false)
	defer teardownRecords(b)

	i := setupIndex(testFolder)
	if err := i.Load(); err != nil {
		b.Fatal(err)
	} else if i.Size() != 0 {
		b.Fatalf("expected %d records, got %d", 0, i.Size())
	}

	for n := 0; n < b.N; n++ {
		_ = i.Create(&testRecord)
	}

	if i.Size() != uint64(b.N) {
		b.Fatalf("expected %d records, found %d", b.N, i.Size())
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

	i.NextId(1)
	if err := i.Create(&testRecord); err != ErrInvalidId {
		t.Fatalf("expected invalid id error, got %v", err)
	}
}
