package storage

import (
	"reflect"
	"testing"

	pb "github.com/evilsocket/sum/proto"
)

func TestRecordDriverMake(t *testing.T) {
	d := RecordDriver{}
	if m := d.Make(); m == nil {
		t.Fatal("unexpected nil message")
	} else if _, ok := m.(*pb.Record); !ok {
		t.Fatalf("unexpected type of record: %v", m)
	}
}

func BenchmarkRecordDriverMake(b *testing.B) {
	d := RecordDriver{}
	for i := 0; i < b.N; i++ {
		if m := d.Make(); m == nil {
			b.Fatal("unexpected nil message")
		}
	}
}

func TestRecordDriverGetID(t *testing.T) {
	d := RecordDriver{}
	m := d.Make()
	if m == nil {
		t.Fatal("unexpected nil message")
	}

	r := m.(*pb.Record)
	r.Id = 666
	if id := d.GetID(m); id != r.Id {
		t.Fatalf("expected id %d, got %d", r.Id, id)
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

func TestRecordDriverSetID(t *testing.T) {
	d := RecordDriver{}
	m := d.Make()
	if m == nil {
		t.Fatal("unexpected nil message")
	}

	d.SetID(m, 666)
	if r := m.(*pb.Record); r.Id != 666 {
		t.Fatalf("expected id %d, got %d", 666, r.Id)
	}
}

func BenchmarkRecordDriverSetID(b *testing.B) {
	d := RecordDriver{}
	m := d.Make()
	if m == nil {
		b.Fatal("unexpected nil message")
	}

	r := m.(*pb.Record)
	for i := 0; i < b.N; i++ {
		id := uint64(i%666) + 1
		d.SetID(m, id)
		if r.Id != id {
			b.Fatalf("expected id %d, got %d", id, r.Id)
		}
	}
}

func TestRecordDriverCopy(t *testing.T) {
	d := RecordDriver{}
	dst := pb.Record{}
	metaSrc := pb.Record{
		Meta: map[string]string{
			"foo":   "bar",
			"hello": "world",
		},
	}
	dataSrc := pb.Record{
		Data: []float32{1, 2, 3, 4, 5, 666},
	}

	if err := d.Copy(&dst, &metaSrc); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(dst.Meta, metaSrc.Meta) {
		t.Fatal("meta values mismatch")
	} else if dst.Data != nil {
		t.Fatal("data field expected as nil")
	} else if err := d.Copy(&dst, &dataSrc); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(dst.Meta, metaSrc.Meta) {
		t.Fatal("meta values mismatch")
	} else if !reflect.DeepEqual(dst.Data, dataSrc.Data) {
		t.Fatal("data vectors mismatch")
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
