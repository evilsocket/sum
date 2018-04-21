package storage

import (
	"testing"

	pb "github.com/evilsocket/sum/proto"
)

func BenchmarkRecordDriverMake(b *testing.B) {
	d := RecordDriver{}
	for i := 0; i < b.N; i++ {
		if m := d.Make(); m == nil {
			b.Fatal("unexpected nil message")
		}
	}
}

func BenchmarkRecordDriverGetID(b *testing.B) {
	d := RecordDriver{}
	m := d.Make()
	if m == nil {
		b.Fatal("unexpected nil message")
	}

	r := m.(*pb.Record)
	for i := 0; i < b.N; i++ {
		r.Id = uint64(i%666) + 1
		if id := d.GetID(m); id != r.Id {
			b.Fatalf("expected id %d, got %d", r.Id, id)
		}
	}
}

func BenchmarkRecordDriverSetID(b *testing.B) {
	d := RecordDriver{}
	m := d.Make()
	if m == nil {
		b.Fatal("unexpected nil message")
	}

	for i := 0; i < b.N; i++ {
		d.SetID(m, uint64(i%666)+1)
	}
}

func BenchmarkRecordDriverCopy(b *testing.B) {
	d := RecordDriver{}
	dst := pb.Record{}
	src := pb.Record{
		Data: []float32{1, 2, 3, 4, 5, 666},
		Meta: map[string]string{
			"foo":   "bar",
			"hello": "world",
		},
	}

	for i := 0; i < b.N; i++ {
		if err := d.Copy(&dst, &src); err != nil {
			b.Fatal(err)
		}
	}
}
