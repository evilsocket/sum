package storage

import (
	"testing"

	pb "github.com/evilsocket/sum/proto"
)

func BenchmarkLoaderLoad(b *testing.B) {
	if err := Flush(&testRecord, testDatFile); err != nil {
		b.Fatal(err)
	}

	var rec pb.Record
	for i := 0; i < b.N; i++ {
		if err := Load(testDatFile, &rec); err != nil {
			b.Fatalf("erorr loading %s: %s", testDatFile, err)
		}
	}
}
