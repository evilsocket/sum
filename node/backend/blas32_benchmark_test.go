package backend

import (
	"math/rand"
	"testing"
	"time"
)

func wrapWithSize(b *testing.B, size int) {
	impl := blas{}
	data := make([]float32, size)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = impl.Wrap(size, data)
	}
}

func dotWithSize(b *testing.B, size int) {
	impl := blas{}
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

func BenchmarkBackendBLAS32Wrap128(b *testing.B) {
	wrapWithSize(b, 128)
}

func BenchmarkBackendBLAS32Wrap256(b *testing.B) {
	wrapWithSize(b, 256)
}

func BenchmarkBackendBLAS32Wrap512(b *testing.B) {
	wrapWithSize(b, 512)
}

func BenchmarkBackendBLAS32Wrap1024(b *testing.B) {
	wrapWithSize(b, 1024)
}

func BenchmarkBackendBLAS32Dot128(b *testing.B) {
	dotWithSize(b, 128)
}

func BenchmarkBackendBLAS32Dot256(b *testing.B) {
	dotWithSize(b, 256)
}

func BenchmarkBackendBLAS32Dot512(b *testing.B) {
	dotWithSize(b, 512)
}

func BenchmarkBackendBLAS32Dot1024(b *testing.B) {
	dotWithSize(b, 1014)
}
