package storage

import (
	"reflect"
	"testing"

	pb "github.com/evilsocket/sum/proto"
)

const (
	testDatFile = "/tmp/storage.test.dat"
)

var (
	testRecord = pb.Record{
		Id:   666,
		Data: []float32{0.6, 0.6, 0.6},
		Meta: map[string]string{"666": "666"},
	}
)

func sameRecord(a, b pb.Record) bool {
	return a.Id == b.Id && reflect.DeepEqual(a.Data, b.Data) && reflect.DeepEqual(a.Meta, b.Meta)
}

func TestStorageFlush(t *testing.T) {
	if err := Flush(&testRecord, testDatFile); err != nil {
		t.Fatal(err)
	}
}

func TestStorageFlushToInvalidPaths(t *testing.T) {
	if err := Flush(nil, "/"); err == nil {
		t.Fatal("error expected for flush with nil message")
	} else if err := Flush(&testRecord, "/"); err == nil {
		t.Fatal("error expected for flush on /")
	}
}

func TestStorageFlushAndBack(t *testing.T) {
	var rec pb.Record
	if err := Flush(&testRecord, testDatFile); err != nil {
		t.Fatal(err)
	} else if err := Load(testDatFile, &rec); err != nil {
		t.Fatal(err)
	} else if !sameRecord(rec, testRecord) {
		t.Fatal("records should be the same")
	}
}
