package wrapper

import (
	"fmt"
	"gonum.org/v1/gonum/mat"
)

type gonum struct {
}

func (backend gonum) Wrap(size int, data []float64) Vector {
	return mat.NewVecDense(size, data)
}

func (backend gonum) Equal(a, b Vector) bool {
	return mat.Equal(a.(mat.Matrix), b.(mat.Matrix))
}

func (backend gonum) EqualApprox(a, b Vector, epsilon float64) bool {
	return mat.EqualApprox(a.(mat.Matrix), b.(mat.Matrix), epsilon)
}

func (backend gonum) Max(a Vector) float64 {
	return mat.Max(a.(mat.Matrix))
}

func (backend gonum) Min(a Vector) float64 {
	return mat.Min(a.(mat.Matrix))
}

func (backend gonum) Sum(a Vector) float64 {
	return mat.Sum(a.(mat.Matrix))
}

func (backend gonum) Dot(a, b Vector) float64 {
	if a == nil {
		fmt.Printf("a=nil b=%v\n", b)
	} else if b == nil {
		fmt.Printf("a=%v b=nil\n", a)
	}

	return mat.Dot(a.(mat.Vector), b.(mat.Vector))
}

func (backend gonum) Det(a Vector) float64 {
	return mat.Det(a.(mat.Matrix))
}
