package wrapper

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/storage"

	"github.com/golang/protobuf/proto"
)

const (
	testFolder  = "/tmp/sum.wrapper.test"
	testRecords = 5
)

var (
	testRecord = pb.Record{
		Id:   666,
		Data: []float32{3, 6, 9},
		Meta: map[string]string{
			"foo":    "bar",
			"some":   "thing",
			"i hate": "tests",
		},
	}
	testShorterRecord = pb.Record{
		Id:   777,
		Data: []float32{1},
		Meta: map[string]string{},
	}
)

func unlink(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func setupRecords(t testing.TB, withValid bool) {
	log.SetOutput(ioutil.Discard)

	// start clean
	teardownRecords(t)

	if err := os.MkdirAll(testFolder, 0755); err != nil {
		t.Fatalf("Error creating %s: %s", testFolder, err)
	}

	if withValid {
		dummy, err := storage.LoadRecords(testFolder)
		if err != nil {
			t.Fatal(err)
		}

		for i := 1; i <= testRecords; i++ {
			if err := dummy.Create(&testRecord); err != nil {
				t.Fatalf("Error creating record: %s", err)
			}
		}
	}
}

func teardownRecords(t testing.TB) {
	if err := unlink(testFolder); err != nil {
		if os.IsNotExist(err) == false {
			t.Fatalf("Error deleting %s: %s", testFolder, err)
		}
	}
}

func TestForRecords(t *testing.T) {
	setupRecords(t, false)
	defer teardownRecords(t)

	records, err := storage.LoadRecords(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	wrapped := ForRecords(records)
	if wrapped.records != records {
		t.Fatal("unexpected records wrapped")
	}
}

func TestFind(t *testing.T) {
	setupRecords(t, true)
	defer teardownRecords(t)

	records, err := storage.LoadRecords(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	wrapped := ForRecords(records)
	for i := 0; i < testRecords; i++ {
		id := uint64(i + 1)
		if r := wrapped.Find(id); r.IsNull() {
			t.Fatalf("wrapped record with id %d not found", id)
		} else if r.Id != id {
			t.Fatalf("expected record with id %d, found %d", id, r.Id)
		}
	}
}

func BenchmarkFind(b *testing.B) {
	setupRecords(b, true)
	defer teardownRecords(b)

	records, err := storage.LoadRecords(testFolder)
	if err != nil {
		b.Fatal(err)
	}

	wrapped := ForRecords(records)
	for i := 0; i < b.N; i++ {
		id := uint64(i%testRecords) + 1
		if r := wrapped.Find(id); r.IsNull() {
			b.Fatalf("wrapped record with id %d not found", id)
		}
	}
}

func TestFindWithInvalidId(t *testing.T) {
	setupRecords(t, false)
	defer teardownRecords(t)

	records, err := storage.LoadRecords(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	wrapped := ForRecords(records)
	for i := 0; i < testRecords; i++ {
		id := uint64(i + 1)
		if r := wrapped.Find(id); r.IsNull() == false {
			t.Fatalf("wrapped record with id %d found, expected none", id)
		}
	}
}

func TestAll(t *testing.T) {
	setupRecords(t, true)
	defer teardownRecords(t)

	records, err := storage.LoadRecords(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	wrapped := ForRecords(records)
	all := wrapped.All()
	if len(all) != testRecords {
		t.Fatalf("expected %d wrapped records, got %d", testRecords, len(all))
	}

	for _, wRec := range all {
		found := false
		records.ForEach(func(m proto.Message) {
			if reflect.DeepEqual(m.(*pb.Record), wRec.record) == true {
				found = true
			}
		})

		if found == false {
			t.Fatalf("record %d not wrapped correctly", wRec.Id)
		}
	}
}

func BenchmarkAll(b *testing.B) {
	setupRecords(b, true)
	defer teardownRecords(b)

	records, err := storage.LoadRecords(testFolder)
	if err != nil {
		b.Fatal(err)
	}
	wrapped := ForRecords(records)
	for i := 0; i < b.N; i++ {
		if all := wrapped.All(); len(all) != testRecords {
			b.Fatalf("expected %d wrapped records, got %d", testRecords, len(all))
		}
	}
}

func TestAllBut(t *testing.T) {
	setupRecords(t, true)
	defer teardownRecords(t)

	records, err := storage.LoadRecords(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	reference := records.Find(1)
	if reference == nil {
		t.Fatal("expected record with id 1")
	}

	wRef := ForRecord(records, reference)
	allBut := ForRecords(records).AllBut(wRef)
	if len(allBut) != testRecords-1 {
		t.Fatalf("expected %d wrapped records, got %d", testRecords-1, len(allBut))
	}

	for _, wRec := range allBut {
		if wRec.Id == reference.Id {
			t.Fatalf("record %d was not supposed to be selected", wRec.Id)
		}

		found := false
		records.ForEach(func(m proto.Message) {
			if reflect.DeepEqual(m.(*pb.Record), wRec.record) == true {
				found = true
			}
		})

		if found == false {
			t.Fatalf("record %d not wrapped correctly", wRec.Id)
		}
	}
}

func BenchmarkAllBut(b *testing.B) {
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

	wrapped := ForRecords(records)
	wRef := ForRecord(records, reference)

	for i := 0; i < b.N; i++ {
		if allBut := wrapped.AllBut(wRef); len(allBut) != testRecords-1 {
			b.Fatalf("expected %d wrapped records, got %d", testRecords-1, len(allBut))
		}
	}
}
