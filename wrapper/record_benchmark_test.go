package wrapper

import (
	"testing"
)

func BenchmarkWrapRecord(b *testing.B) {
	for i := 0; i < b.N; i++ {
		wrapped := WrapRecord(nil, &testRecord)
		if wrapped.ID != testRecord.Id {
			b.Fatalf("expected record with id %d, %d found", testRecord.Id, wrapped.ID)
		}
	}
}

func BenchmarkWrappedRecordIs(b *testing.B) {
	a := WrapRecord(nil, &testRecord)
	c := WrapRecord(nil, &testRecord)

	for i := 0; i < b.N; i++ {
		if !a.Is(c) {
			b.Fatal("records should match")
		}
	}
}

func BenchmarkWrappedRecordGet(b *testing.B) {
	r := WrapRecord(nil, &testRecord)
	idx := 0
	v := testRecord.Data[idx]

	for i := 0; i < b.N; i++ {
		if r.Get(idx) != v {
			b.Fatalf("expected value %f at index %d, got %f", v, idx, r.Get(idx))
		}
	}
}

func BenchmarkWrappedRecordSet(b *testing.B) {
	r := WrapRecord(nil, &testRecord)
	max := len(testRecord.Data)

	for i := 0; i < b.N; i++ {
		r.Set(i%max, 3.14)
	}
}

func BenchmarkWrappedRecordMeta(b *testing.B) {
	r := WrapRecord(nil, &testRecord)
	for i := 0; i < b.N; i++ {
		if got := r.Meta("foo"); got != "bar" {
			b.Fatalf("expecting '%s' for meta '%s', got '%s'", "bar", "foot", got)
		}
	}
}

func BenchmarkWrappedRecordSetMeta(b *testing.B) {
	r := WrapRecord(nil, &testRecord)
	k := "new"
	v := "meta value"
	for i := 0; i < b.N; i++ {
		r.SetMeta(k, v)
	}
}

func BenchmarkWrappedRecordDot(b *testing.B) {
	testRecord.Data = []float32{3, 6, 9}
	shouldBe := 126.0

	a := WrapRecord(nil, &testRecord)
	c := WrapRecord(nil, &testRecord)

	for i := 0; i < b.N; i++ {
		if dot := a.Dot(c); dot != shouldBe {
			b.Fatalf("dot product should be %f, got %f", shouldBe, dot)
		}
	}
}

func BenchmarkWrappedRecordMagnitude(b *testing.B) {
	testRecord.Data = []float32{0, 0, 2}
	shouldBe := 2.0
	a := WrapRecord(nil, &testRecord)

	for i := 0; i < b.N; i++ {
		if mag := a.Magnitude(); mag != shouldBe {
			b.Fatalf("magnitude should be %f, got %f", shouldBe, mag)
		}
	}
}

func BenchmarkWrappedRecordCosine(b *testing.B) {
	testRecord.Data = []float32{3, 6, 9}
	a := WrapRecord(nil, &testRecord)
	c := WrapRecord(nil, &testRecord)

	for i := 0; i < b.N; i++ {
		if cos := a.Cosine(c); cos != 1.0 {
			b.Fatalf("cosine similarity should be %f, got %f", 1.0, cos)
		}
	}
}
