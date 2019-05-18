package wrapper

import (
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	. "github.com/stretchr/testify/require"

	pb "github.com/evilsocket/sum/proto"

	"github.com/evilsocket/islazy/log"
)

func init() {
	log.Level = log.ERROR
}

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

func TestWrappedRecordEqual(t *testing.T) {
	a := WrapRecord(&testRecord)
	b := WrapRecord(&testShorterRecord)
	if a.Equal(a) == false {
		t.Fatalf("expecting true")
	} else if a.Equal(b) || b.Equal(a) {
		t.Fatalf("expecting false")
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

func TestWrappedRecordDotRange(t *testing.T) {
	testRecord.Data = []float32{3, 6, 9, 1, 2, 3, 4, 5, 666}
	shouldBe := 126.0

	a := WrapRecord(&testRecord)
	b := WrapRecord(&testRecord)

	if dot := a.DotRange(b, 0, 3); dot != shouldBe {
		t.Fatalf("dot product should be %f, got %f", shouldBe, dot)
	}
}

func TestWrappedRecordDotSub(t *testing.T) {
	testRecord.Data = []float32{3, 6, 9, 1, 2, 3, 4, 5, 666}
	shouldBe := 126.0

	a := WrapRecord(&testRecord)
	b := WrapRecord(&testRecord)

	if dot := a.DotSub(b, 3); dot != shouldBe {
		t.Fatalf("dot product should be %f, got %f", shouldBe, dot)
	}
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

func TestSerialization(t *testing.T) {
	r := &pb.Record{Id: 1, Meta: map[string]string{"key": "value"}, Data: []float32{0.1, 0.2, 0.3}}

	str, err := RecordToCompressedText(r)
	Nil(t, err)
	r1, err := FromCompressedText(str)
	Nil(t, err)
	False(t, r1.IsNull())
	True(t, proto.Equal(r, r1.record))

	r = nil

	str, err = RecordToCompressedText(r)
	Nil(t, err)
	r1, err = FromCompressedText(str)
	Nil(t, err)
	Nil(t, r1.record)
	True(t, r1.IsNull())
}
