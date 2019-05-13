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
	} else if _, ok := m.(*pb.Oracle); !ok {
		t.Fatalf("unexpected type of record: %v", m)
	}
}

func TestOracleDriverGetID(t *testing.T) {
	d := OracleDriver{}
	m := d.Make()
	if m == nil {
		t.Fatal("unexpected nil message")
	}

	r := m.(*pb.Oracle)
	r.Id = 666
	if id := d.GetID(m); id != r.Id {
		t.Fatalf("expected id %d, got %d", r.Id, id)
	}
}

func TestOracleDriverSetID(t *testing.T) {
	d := OracleDriver{}
	m := d.Make()
	if m == nil {
		t.Fatal("unexpected nil message")
	}

	d.SetID(m, 666)
	if r := m.(*pb.Oracle); r.Id != 666 {
		t.Fatalf("expected id %d, got %d", 666, r.Id)
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
	} else if !reflect.DeepEqual(dst, src) {
		t.Fatal("contents mismatch")
	}
}
