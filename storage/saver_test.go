package storage

import (
	"reflect"
	"testing"

	pb "github.com/evilsocket/sum/proto"
)

const (
	testDatFile = "/tmp/testflush.dat"
)

var (
	testRecord = pb.Record{
		Id:   666,
		Data: []float32{0.6, 0.6, 0.6},
		Meta: map[string]string{"666": "666"},
	}
)

func TestFlush(t *testing.T) {
	if err := Flush(&testRecord, testDatFile); err != nil {
		t.Fatal(err)
	}
}

func TestFlushWithError(t *testing.T) {
	if err := Flush(&testRecord, "/"); err == nil {
		t.Fatal("wasn't supposed to happen")
	}
}

func TestFlushAndBack(t *testing.T) {
	if err := Flush(&testRecord, testDatFile); err != nil {
		t.Fatal(err)
	}

	var rec pb.Record
	if err := Load(testDatFile, &rec); err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(rec, testRecord) == false {
		t.Fatal("records should be the same")
	}
}

func BenchmarkFlush(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if err := Flush(&testRecord, testDatFile); err != nil {
			b.Fatal(err)
		}
	}
}
