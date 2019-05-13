package storage

import (
	"testing"

	pb "github.com/evilsocket/sum/proto"
)

func BenchmarkOracleDriverMake(b *testing.B) {
	d := OracleDriver{}
	for i := 0; i < b.N; i++ {
		if m := d.Make(); m == nil {
			b.Fatal("unexpected nil message")
		}
	}
}

func BenchmarkOracleDriverGetID(b *testing.B) {
	d := OracleDriver{}
	m := d.Make()
	if m == nil {
		b.Fatal("unexpected nil message")
	}

	r := m.(*pb.Oracle)
	for i := 0; i < b.N; i++ {
		r.Id = uint64(i%666) + 1
		if id := d.GetID(m); id != r.Id {
			b.Fatalf("expected id %d, got %d", r.Id, id)
		}
	}
}

func BenchmarkOracleDriverSetID(b *testing.B) {
	d := OracleDriver{}
	m := d.Make()
	if m == nil {
		b.Fatal("unexpected nil message")
	}

	for i := 0; i < b.N; i++ {
		d.SetID(m, uint64(i%666)+1)
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
