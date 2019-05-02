package wrapper

import (
	"reflect"
	"testing"

	pb "github.com/evilsocket/sum/proto"
)

func assertPanic(t *testing.T, msg string, f func()) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal(msg)
		}
	}()
	f()
}

func TestWrapRecord(t *testing.T) {
	wrapped := WrapRecord(&testRecord)
	if wrapped.ID != testRecord.Id {
		t.Fatalf("expected record with id %d, %d found", testRecord.Id, wrapped.ID)
	} else if !reflect.DeepEqual(*wrapped.record, testRecord) {
		t.Fatal("unexpected wrapped record")
	}
}

func TestWrapRecordWithNil(t *testing.T) {
	wrapped := WrapRecord(nil)
	if !wrapped.IsNull() {
		t.Fatal("expected null wrapped")
	}
}

func TestWrappedRecordIs(t *testing.T) {
	a := WrapRecord(&testRecord)
	b := WrapRecord(&testRecord)
	c := WrapRecord(nil)

	if !a.Is(b) {
		t.Fatal("records should match")
	} else if !b.Is(a) {
		t.Fatal("records should match")
	} else if a.Is(c) {
		t.Fatal("records should not match")
	} else if b.Is(c) {
		t.Fatal("records should not match")
	} else if c.Is(b) {
		t.Fatal("records should not match")
	}
}

func TestWrappedRecordGet(t *testing.T) {
	r := WrapRecord(&testRecord)
	for idx, v := range testRecord.Data {
		if r.Get(idx) != v {
			t.Fatalf("expected value %f at index %d, got %f", v, idx, r.Get(idx))
		}
	}
}

func TestWrappedRecordGetWithInvalidIndex(t *testing.T) {
	assertPanic(t, "access to an invalid index should panic", func() {
		WrapRecord(&testRecord).Get(666)
	})
}

func TestWrappedRecordMeta(t *testing.T) {
	r := WrapRecord(&testRecord)
	for k, v := range testRecord.Meta {
		if got := r.Meta(k); got != v {
			t.Fatalf("expecting '%s' for meta '%s', got '%s'", v, k, got)
		}
	}
}

func TestWrappedRecordMetaWithInvalidKey(t *testing.T) {
	r := WrapRecord(&testRecord)
	if got := r.Meta("i do not exist"); got != "" {
		t.Fatalf("expecting empty value, got '%s'", got)
	}
}

func TestWrappedRecordDot(t *testing.T) {
	testRecord.Data = []float32{3, 6, 9}
	shouldBe := 126.0

	a := WrapRecord(&testRecord)
	b := WrapRecord(&testRecord)

	if dot := a.Dot(b); dot != shouldBe {
		t.Fatalf("dot product should be %f, got %f", shouldBe, dot)
	}
}

func TestWrappedRecordDotWithNull(t *testing.T) {
	assertPanic(t, "dot product should panic with null wrapped record", func() {
		WrapRecord(nil).Dot(WrapRecord(nil))
	})

	assertPanic(t, "dot product should panic with null wrapped record", func() {
		WrapRecord(&testRecord).Dot(WrapRecord(nil))
	})

	assertPanic(t, "dot product should panic with null wrapped record", func() {
		WrapRecord(nil).Dot(WrapRecord(&testRecord))
	})
}

func TestWrappedRecordDotWithIncompatibleSizes(t *testing.T) {
	assertPanic(t, "dot product should panic with vectors of different sizes", func() {
		WrapRecord(&testRecord).Dot(WrapRecord(&testShorterRecord))
	})
}

func TestWrappedRecordMagnitude(t *testing.T) {
	testRecord.Data = []float32{0, 0, 2}
	shouldBe := 2.0
	a := WrapRecord(&testRecord)
	if mag := a.Magnitude(); mag != shouldBe {
		t.Fatalf("magnitude should be %f, got %f", shouldBe, mag)
	}
}

func TestWrappedRecordMagnitudeWithNull(t *testing.T) {
	assertPanic(t, "magnitude product should panic with null wrapped record", func() {
		_ = WrapRecord(nil).Magnitude()
	})
}

func TestWrappedRecordCosine(t *testing.T) {
	a := WrapRecord(&pb.Record{Data: []float32{3, 6, 9}})
	b := WrapRecord(&pb.Record{Data: []float32{0, 0, 0}})
	if cos := a.Cosine(b); cos != 0.0 {
		t.Fatalf("cosine similarity should be %f, got %f", 0.0, cos)
	}

	b.SetData(a.record.Data)
	if cos := a.Cosine(b); cos != 1.0 {
		t.Fatalf("cosine similarity should be %f, got %f", 1.0, cos)
	}
}

func TestWrappedRecordCosineWithNull(t *testing.T) {
	assertPanic(t, "cosine similarity should panic with null wrapped record", func() {
		WrapRecord(nil).Cosine(WrapRecord(nil))
	})

	assertPanic(t, "cosine similarity should panic with null wrapped record", func() {
		WrapRecord(&testRecord).Cosine(WrapRecord(nil))
	})

	assertPanic(t, "cosine similarity should panic with null wrapped record", func() {
		WrapRecord(nil).Cosine(WrapRecord(&testRecord))
	})
}

func TestWrappedRecordCosineWithIncompatibleSizes(t *testing.T) {
	assertPanic(t, "cosine similarity should panic with vectors of different sizes", func() {
		WrapRecord(&testRecord).Cosine(WrapRecord(&testShorterRecord))
	})
}
