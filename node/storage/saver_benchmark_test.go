package storage

import (
	"testing"
)

func BenchmarkStorageFlush(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if err := Flush(&testRecord, testDatFile); err != nil {
			b.Fatal(err)
		}
	}
}
