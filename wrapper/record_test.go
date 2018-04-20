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

func TestForRecord(t *testing.T) {
	wrapped := ForRecord(nil, &testRecord)
	if wrapped.Id != testRecord.Id {
		t.Fatalf("expected record with id %d, %d found", testRecord.Id, wrapped.Id)
	} else if wrapped.store != nil {
		t.Fatal("unexpected store pointer")
	} else if reflect.DeepEqual(*wrapped.record, testRecord) == false {
		t.Fatal("unexpected wrapped record")
	}
}

func BenchmarkForRecord(b *testing.B) {
	for i := 0; i < b.N; i++ {
		wrapped := ForRecord(nil, &testRecord)
		if wrapped.Id != testRecord.Id {
			b.Fatalf("expected record with id %d, %d found", testRecord.Id, wrapped.Id)
		}
	}
}

func TestForRecordWithNil(t *testing.T) {
	wrapped := ForRecord(nil, nil)
	if wrapped.IsNull() == false {
		t.Fatal("expected null wrapped")
	}
}

func TestIs(t *testing.T) {
	a := ForRecord(nil, &testRecord)
	b := ForRecord(nil, &testRecord)
	c := ForRecord(nil, nil)

	if a.Is(b) == false {
		t.Fatal("records should match")
	} else if b.Is(a) == false {
		t.Fatal("records should match")
	} else if a.Is(c) == true {
		t.Fatal("records should not match")
	} else if b.Is(c) == true {
		t.Fatal("records should not match")
	} else if c.Is(b) == true {
		t.Fatal("records should not match")
	}
}

func BenchmarkIs(b *testing.B) {
	a := ForRecord(nil, &testRecord)
	c := ForRecord(nil, &testRecord)

	for i := 0; i < b.N; i++ {
		if a.Is(c) == false {
			b.Fatal("records should match")
		}
	}
}

func TestGet(t *testing.T) {
	r := ForRecord(nil, &testRecord)
	for idx, v := range testRecord.Data {
		if r.Get(idx) != v {
			t.Fatalf("expected value %f at index %d, got %f", v, idx, r.Get(idx))
		}
	}
}

func BenchmarkGet(b *testing.B) {
	r := ForRecord(nil, &testRecord)
	idx := 0
	v := testRecord.Data[idx]

	for i := 0; i < b.N; i++ {
		if r.Get(idx) != v {
			b.Fatalf("expected value %f at index %d, got %f", v, idx, r.Get(idx))
		}
	}
}

func TestGetWithInvalidIndex(t *testing.T) {
	assertPanic(t, "access to an invalid index should panic", func() {
		ForRecord(nil, &testRecord).Get(666)
	})
}

func TestSet(t *testing.T) {
	r := ForRecord(nil, &testRecord)
	for idx, v := range testRecord.Data {
		new := v * 3.14
		r.Set(idx, new)
		if r.Get(idx) != new {
			t.Fatalf("expected new value %f at index %d, got %f", new, idx, r.Get(idx))
		}
	}
}

func BenchmarkSet(b *testing.B) {
	r := ForRecord(nil, &testRecord)
	max := len(testRecord.Data)
	for i := 0; i < b.N; i++ {
		idx := i % max
		r.Set(idx, 3.14)
	}
}

func TestSetWithInvalidIndex(t *testing.T) {
	assertPanic(t, "access to an invalid index should panic", func() {
		ForRecord(nil, &testRecord).Set(666, 3.14)
	})
}

func TestSetWithStore(t *testing.T) {
	setupRecords(t, true)
	defer teardownRecords(t)

	records, err := storage.LoadRecords(testFolder)
	if err != nil {
		t.Fatal(err)
	}
	stored := records.Find(1)
	r := ForRecord(records, stored)
	for idx, v := range testRecord.Data {
		new := v * 3.14
		r.Set(idx, new)
		if r.Get(idx) != new {
			t.Fatalf("expected new value %f at index %d, got %f", new, idx, r.Get(idx))
		}
	}
}

func TestSetWithStoreAndInvalidId(t *testing.T) {
	setupRecords(t, false)
	defer teardownRecords(t)

	records, err := storage.LoadRecords(testFolder)
	if err != nil {
		t.Fatal(err)
	}
	r := ForRecord(records, &testRecord)
	for idx, v := range testRecord.Data {
		new := v * 3.14
		r.Set(idx, new)
		if r.Get(idx) == new {
			t.Fatal("expected old value to be unchanged")
		}
	}
}

func TestMeta(t *testing.T) {
	r := ForRecord(nil, &testRecord)
	for k, v := range testRecord.Meta {
		if got := r.Meta(k); got != v {
			t.Fatalf("expecting '%s' for meta '%s', got '%s'", v, k, got)
		}
	}
}

func BenchmarkMeta(b *testing.B) {
	r := ForRecord(nil, &testRecord)
	for i := 0; i < b.N; i++ {
		for k, v := range testRecord.Meta {
			if got := r.Meta(k); got != v {
				b.Fatalf("expecting '%s' for meta '%s', got '%s'", v, k, got)
			}
		}
	}
}

func TestMetaWithInvalidKey(t *testing.T) {
	r := ForRecord(nil, &testRecord)
	if got := r.Meta("i do not exist"); got != "" {
		t.Fatalf("expecting empty value, got '%s'", got)
	}
}

func TestDot(t *testing.T) {
	testRecord.Data = []float32{3, 6, 9}
	shouldBe := 126.0

	a := ForRecord(nil, &testRecord)
	b := ForRecord(nil, &testRecord)

	if dot := a.Dot(b); dot != shouldBe {
		t.Fatalf("dot product should be %f, got %f", shouldBe, dot)
	}
}

func BenchmarkDot(b *testing.B) {
	testRecord.Data = []float32{3, 6, 9}
	shouldBe := 126.0

	a := ForRecord(nil, &testRecord)
	c := ForRecord(nil, &testRecord)

	for i := 0; i < b.N; i++ {
		dot := a.Dot(c)
		if dot != shouldBe {
			b.Fatalf("dot product should be %f, got %f", shouldBe, dot)
		}
	}
}

func TestDotWithNull(t *testing.T) {
	assertPanic(t, "dot product should panic with null wrapped record", func() {
		ForRecord(nil, nil).Dot(ForRecord(nil, nil))
	})

	assertPanic(t, "dot product should panic with null wrapped record", func() {
		ForRecord(nil, &testRecord).Dot(ForRecord(nil, nil))
	})

	assertPanic(t, "dot product should panic with null wrapped record", func() {
		ForRecord(nil, nil).Dot(ForRecord(nil, &testRecord))
	})
}

func TestDotWithIncompatibleSizes(t *testing.T) {
	assertPanic(t, "dot product should panic with vectors of different sizes", func() {
		ForRecord(nil, &testRecord).Dot(ForRecord(nil, &testShorterRecord))
	})
}

func TestMagnitude(t *testing.T) {
	testRecord.Data = []float32{0, 0, 2}
	shouldBe := 2.0
	a := ForRecord(nil, &testRecord)
	if mag := a.Magnitude(); mag != shouldBe {
		t.Fatalf("magnitude should be %f, got %f", shouldBe, mag)
	}
}

func BenchmarkMagnitude(b *testing.B) {
	testRecord.Data = []float32{0, 0, 2}
	shouldBe := 2.0
	a := ForRecord(nil, &testRecord)

	for i := 0; i < b.N; i++ {
		if mag := a.Magnitude(); mag != shouldBe {
			b.Fatalf("magnitude should be %f, got %f", shouldBe, mag)
		}
	}
}

func TestMagnitudeWithNull(t *testing.T) {
	assertPanic(t, "magnitude product should panic with null wrapped record", func() {
		_ = ForRecord(nil, nil).Magnitude()
	})
}

func TestCosine(t *testing.T) {
	a := ForRecord(nil, &pb.Record{Data: []float32{3, 6, 9}})
	b := ForRecord(nil, &pb.Record{Data: []float32{0, 0, 0}})
	if cos := a.Cosine(b); cos != 0.0 {
		t.Fatalf("cosine similarity should be %f, got %f", 0.0, cos)
	}

	b.record.Data = a.record.Data
	if cos := a.Cosine(b); cos != 1.0 {
		t.Fatalf("cosine similarity should be %f, got %f", 1.0, cos)
	}
}

func BenchmarkCosine(b *testing.B) {
	testRecord.Data = []float32{3, 6, 9}
	a := ForRecord(nil, &testRecord)
	c := ForRecord(nil, &testRecord)
	for i := 0; i < b.N; i++ {
		cos := a.Cosine(c)
		if cos != 1.0 {
			b.Fatalf("cosine similarity should be %f, got %f", 1.0, cos)
		}
	}
}

func TestCosineWithNull(t *testing.T) {
	assertPanic(t, "cosine similarity should panic with null wrapped record", func() {
		ForRecord(nil, nil).Cosine(ForRecord(nil, nil))
	})

	assertPanic(t, "cosine similarity should panic with null wrapped record", func() {
		ForRecord(nil, &testRecord).Cosine(ForRecord(nil, nil))
	})

	assertPanic(t, "cosine similarity should panic with null wrapped record", func() {
		ForRecord(nil, nil).Cosine(ForRecord(nil, &testRecord))
	})
}

func TestCosineWithIncompatibleSizes(t *testing.T) {
	assertPanic(t, "cosine similarity should panic with vectors of different sizes", func() {
		ForRecord(nil, &testRecord).Cosine(ForRecord(nil, &testShorterRecord))
	})
}
