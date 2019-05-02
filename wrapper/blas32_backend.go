package wrapper

import (
	"gonum.org/v1/gonum/blas/blas32"
)

type blas struct {
}

type blasWrap struct {
	v  blas32.Vector
	sz int
}

func (backend blas) Wrap(size int, data []float32) Vector {
	return blasWrap{
		v: blas32.Vector{
			Inc:  1,
			Data: data,
		},
		sz: size,
	}
}

func (backend blas) Dot(a, b Vector) float64 {
	return float64(blas32.Dot(a.(blasWrap).sz, a.(blasWrap).v, b.(blasWrap).v))
}
