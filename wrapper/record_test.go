package wrapper

import (
	"reflect"
	"testing"

	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/storage"
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
	wrapped := WrapRecord(nil, &testRecord)
	if wrapped.ID != testRecord.Id {
		t.Fatalf("expected record with id %d, %d found", testRecord.Id, wrapped.ID)
	} else if wrapped.store != nil {
		t.Fatal("unexpected store pointer")
	} else if !reflect.DeepEqual(*wrapped.record, testRecord) {
		t.Fatal("unexpected wrapped record")
	}
}

func TestWrapRecordWithNil(t *testing.T) {
	wrapped := WrapRecord(nil, nil)
	if !wrapped.IsNull() {
		t.Fatal("expected null wrapped")
	}
}

func TestWrappedRecordIs(t *testing.T) {
	a := WrapRecord(nil, &testRecord)
	b := WrapRecord(nil, &testRecord)
	c := WrapRecord(nil, nil)

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
	r := WrapRecord(nil, &testRecord)
	for idx, v := range testRecord.Data {
		if r.Get(idx) != v {
			t.Fatalf("expected value %f at index %d, got %f", v, idx, r.Get(idx))
		}
	}
}

func TestWrappedRecordGetWithInvalidIndex(t *testing.T) {
	assertPanic(t, "access to an invalid index should panic", func() {
		WrapRecord(nil, &testRecord).Get(666)
	})
}

func TestWrappedRecordSet(t *testing.T) {
	r := WrapRecord(nil, &testRecord)
	for idx, v := range testRecord.Data {
		new := v * 3.14
		r.Set(idx, new)
		if r.Get(idx) != new {
			t.Fatalf("expected new value %f at index %d, got %f", new, idx, r.Get(idx))
		}
	}
}

func TestWrappedRecordSetWithInvalidIndex(t *testing.T) {
	assertPanic(t, "access to an invalid index should panic", func() {
		WrapRecord(nil, &testRecord).Set(666, 3.14)
	})
}

func TestWrappedRecordSetWithStore(t *testing.T) {
	setupRecords(t, true)
	defer teardownRecords(t)

	records, err := storage.LoadRecords(testFolder)
	if err != nil {
		t.Fatal(err)
	}
	stored := records.Find(1)
	r := WrapRecord(records, stored)
	for idx, v := range testRecord.Data {
		new := v * 3.14
		r.Set(idx, new)
		if r.Get(idx) != new {
			t.Fatalf("expected new value %f at index %d, got %f", new, idx, r.Get(idx))
		}
	}
}

func TestWrappedRecordSetWithStoreAndInvalidId(t *testing.T) {
	setupRecords(t, false)
	defer teardownRecords(t)

	records, err := storage.LoadRecords(testFolder)
	if err != nil {
		t.Fatal(err)
	}
	r := WrapRecord(records, &testRecord)
	for idx, v := range testRecord.Data {
		new := v * 3.14
		r.Set(idx, new)
		if r.Get(idx) == new {
			t.Fatal("expected old value to be unchanged")
		}
	}
}

func TestWrappedRecordMeta(t *testing.T) {
	r := WrapRecord(nil, &testRecord)
	for k, v := range testRecord.Meta {
		if got := r.Meta(k); got != v {
			t.Fatalf("expecting '%s' for meta '%s', got '%s'", v, k, got)
		}
	}
}

func TestWrappedRecordMetaWithInvalidKey(t *testing.T) {
	r := WrapRecord(nil, &testRecord)
	if got := r.Meta("i do not exist"); got != "" {
		t.Fatalf("expecting empty value, got '%s'", got)
	}
}

func TestWrappedRecordSetMeta(t *testing.T) {
	r := WrapRecord(nil, &testRecord)
	k := "new"
	v := "meta value"
	r.SetMeta(k, v)
	if got := r.Meta(k); got != v {
		t.Fatalf("expecting '%s' for meta '%s', got '%s'", v, k, got)
	}
}

func TestWrappedRecordSetMetaWithInvalidId(t *testing.T) {
	setupRecords(t, false)
	defer teardownRecords(t)

	records, err := storage.LoadRecords(testFolder)
	if err != nil {
		t.Fatal(err)
	}
	r := WrapRecord(records, &testRecord)
	for k, v := range testRecord.Meta {
		newValue := v + " changed"
		r.SetMeta(k, newValue)
		if got := r.Meta(k); got == newValue {
			t.Fatal("expecting old meta value to be unchanged")
		}
	}
}

func TestWrappedRecordDot(t *testing.T) {
	testRecord.Data = []float32{3, 6, 9}
	shouldBe := 126.0

	a := WrapRecord(nil, &testRecord)
	b := WrapRecord(nil, &testRecord)

	if dot := a.Dot(b); dot != shouldBe {
		t.Fatalf("dot product should be %f, got %f", shouldBe, dot)
	}
}

func TestWrappedRecordDotWithNull(t *testing.T) {
	assertPanic(t, "dot product should panic with null wrapped record", func() {
		WrapRecord(nil, nil).Dot(WrapRecord(nil, nil))
	})

	assertPanic(t, "dot product should panic with null wrapped record", func() {
		WrapRecord(nil, &testRecord).Dot(WrapRecord(nil, nil))
	})

	assertPanic(t, "dot product should panic with null wrapped record", func() {
		WrapRecord(nil, nil).Dot(WrapRecord(nil, &testRecord))
	})
}

func TestWrappedRecordDotWithIncompatibleSizes(t *testing.T) {
	assertPanic(t, "dot product should panic with vectors of different sizes", func() {
		WrapRecord(nil, &testRecord).Dot(WrapRecord(nil, &testShorterRecord))
	})
}

func TestWrappedRecordMagnitude(t *testing.T) {
	testRecord.Data = []float32{0, 0, 2}
	shouldBe := 2.0
	a := WrapRecord(nil, &testRecord)
	if mag := a.Magnitude(); mag != shouldBe {
		t.Fatalf("magnitude should be %f, got %f", shouldBe, mag)
	}
}

func TestWrappedRecordMagnitudeWithNull(t *testing.T) {
	assertPanic(t, "magnitude product should panic with null wrapped record", func() {
		_ = WrapRecord(nil, nil).Magnitude()
	})
}

func TestWrappedRecordCosine(t *testing.T) {
	a := WrapRecord(nil, &pb.Record{Data: []float32{3, 6, 9}})
	b := WrapRecord(nil, &pb.Record{Data: []float32{0, 0, 0}})
	if cos := a.Cosine(b); cos != 0.0 {
		t.Fatalf("cosine similarity should be %f, got %f", 0.0, cos)
	}

	b.record.Data = a.record.Data
	if cos := a.Cosine(b); cos != 1.0 {
		t.Fatalf("cosine similarity should be %f, got %f", 1.0, cos)
	}
}

func TestWrappedRecordCosineWithNull(t *testing.T) {
	assertPanic(t, "cosine similarity should panic with null wrapped record", func() {
		WrapRecord(nil, nil).Cosine(WrapRecord(nil, nil))
	})

	assertPanic(t, "cosine similarity should panic with null wrapped record", func() {
		WrapRecord(nil, &testRecord).Cosine(WrapRecord(nil, nil))
	})

	assertPanic(t, "cosine similarity should panic with null wrapped record", func() {
		WrapRecord(nil, nil).Cosine(WrapRecord(nil, &testRecord))
	})
}

func TestWrappedRecordCosineWithIncompatibleSizes(t *testing.T) {
	assertPanic(t, "cosine similarity should panic with vectors of different sizes", func() {
		WrapRecord(nil, &testRecord).Cosine(WrapRecord(nil, &testShorterRecord))
	})
}
