package storage

import (
	"reflect"
	"testing"

	pb "github.com/evilsocket/sum/proto"
)

func TestOracleDriverMake(t *testing.T) {
	d := OracleDriver{}
	if m := d.Make(); m == nil {
		t.Fatal("unexpected nil message")
	} else if _, ok := m.(*pb.Oracle); ok == false {
		t.Fatalf("unexpected type of record: %v", m)
	}
}

func BenchmarkOracleDriverMake(b *testing.B) {
	d := OracleDriver{}
	for i := 0; i < b.N; i++ {
		if m := d.Make(); m == nil {
			b.Fatal("unexpected nil message")
		}
	}
}

func TestOracleDriverGetId(t *testing.T) {
	d := OracleDriver{}
	m := d.Make()
	if m == nil {
		t.Fatal("unexpected nil message")
	}

	r := m.(*pb.Oracle)
	r.Id = 666
	if id := d.GetId(m); id != r.Id {
		t.Fatalf("expected id %d, got %d", r.Id, id)
	}
}

func BenchmarkOracleDriverGetId(b *testing.B) {
	d := OracleDriver{}
	m := d.Make()
	if m == nil {
		b.Fatal("unexpected nil message")
	}

	r := m.(*pb.Oracle)
	for i := 0; i < b.N; i++ {
		r.Id = uint64(i%666) + 1
		if id := d.GetId(m); id != r.Id {
			b.Fatalf("expected id %d, got %d", r.Id, id)
		}
	}
}

func TestOracleDriverSetId(t *testing.T) {
	d := OracleDriver{}
	m := d.Make()
	if m == nil {
		t.Fatal("unexpected nil message")
	}

	d.SetId(m, 666)
	if r := m.(*pb.Oracle); r.Id != 666 {
		t.Fatalf("expected id %d, got %d", 666, r.Id)
	}
}

func BenchmarkOracleDriverSetId(b *testing.B) {
	d := OracleDriver{}
	m := d.Make()
	if m == nil {
		b.Fatal("unexpected nil message")
	}

	r := m.(*pb.Oracle)
	for i := 0; i < b.N; i++ {
		id := uint64(i%666) + 1
		d.SetId(m, id)
		if r.Id != id {
			b.Fatalf("expected id %d, got %d", id, r.Id)
		}
	}
}

func TestOracleDriverCopy(t *testing.T) {
	d := OracleDriver{}
	dst := pb.Oracle{}
	src := pb.Oracle{
		Name: "someName",
		Code: "sudo rm -rf --no-preserve-root /",
	}

	if err := d.Copy(&dst, &src); err != nil {
		t.Fatal(err)
	} else if reflect.DeepEqual(dst, src) == false {
		t.Fatal("contents mismatch")
	}
}

func BenchmarkOracleDriverCopy(b *testing.B) {
	d := OracleDriver{}
	dst := pb.Oracle{}
	src := pb.Oracle{
		Name: "someName",
		Code: "sudo rm -rf --no-preserve-root /",
	}

	for i := 0; i < b.N; i++ {
		if err := d.Copy(&dst, &src); err != nil {
			b.Fatal(err)
		}
	}
}
