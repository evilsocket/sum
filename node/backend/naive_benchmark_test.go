package backend

import (
	"math/rand"
	"testing"
	"time"
)

func wrapWithSize(impl implementation, b *testing.B, size int) {
	data := make([]float32, size)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = impl.Wrap(size, data)
	}
}

func dotWithSize(impl implementation, b *testing.B, size int) {
	adata := make([]float32, size)
	bdata := make([]float32, size)

	s := rand.NewSource(time.Now().Unix())
	r := rand.New(s)

	for i := 0; i < size; i++ {
		adata[i] = r.Float32()
		bdata[i] = r.Float32()
	}

	va := impl.Wrap(size, adata)
	vb := impl.Wrap(size, bdata)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = impl.Dot(va, vb)
	}
}

func BenchmarkBackendNaiveWrap128(b *testing.B) {
	wrapWithSize(naive{}, b, 128)
}

func BenchmarkBackendNaiveWrap256(b *testing.B) {
	wrapWithSize(naive{}, b, 256)
}

func BenchmarkBackendNaiveWrap512(b *testing.B) {
	wrapWithSize(naive{}, b, 512)
}

func BenchmarkBackendNaiveWrap1024(b *testing.B) {
	wrapWithSize(naive{}, b, 1024)
}

func BenchmarkBackendNaiveDot128(b *testing.B) {
	dotWithSize(naive{}, b, 128)
}

func BenchmarkBackendNaiveDot256(b *testing.B) {
	dotWithSize(naive{}, b, 256)
}

func BenchmarkBackendNaiveDot512(b *testing.B) {
	dotWithSize(naive{}, b, 512)
}

func BenchmarkBackendNaiveDot1024(b *testing.B) {
	dotWithSize(naive{}, b, 1014)
}
