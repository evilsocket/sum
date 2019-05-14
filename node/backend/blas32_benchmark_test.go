package backend

import (
	"testing"
)

func BenchmarkBackendBLAS32Wrap128(b *testing.B) {
	wrapWithSize(blas{}, b, 128)
}

func BenchmarkBackendBLAS32Wrap256(b *testing.B) {
	wrapWithSize(blas{}, b, 256)
}

func BenchmarkBackendBLAS32Wrap512(b *testing.B) {
	wrapWithSize(blas{}, b, 512)
}

func BenchmarkBackendBLAS32Wrap1024(b *testing.B) {
	wrapWithSize(blas{}, b, 1024)
}

func BenchmarkBackendBLAS32Dot128(b *testing.B) {
	dotWithSize(blas{}, b, 128)
}

func BenchmarkBackendBLAS32Dot256(b *testing.B) {
	dotWithSize(blas{}, b, 256)
}

func BenchmarkBackendBLAS32Dot512(b *testing.B) {
	dotWithSize(blas{}, b, 512)
}

func BenchmarkBackendBLAS32Dot1024(b *testing.B) {
	dotWithSize(blas{}, b, 1014)
}
