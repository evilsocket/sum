package backend

// Vector is an opaque interface to whatever the specific backend implementation
// will return as object wrapper.
type Vector interface{}

type implementation interface {
	Name() string

	Wrap(size int, data []float32) Vector

	Dot(a, b Vector) float64
}

// TODO: pick at runtime the best backend available ( CUDA, OpenCL or BLAS32 ).
var impl = blas{}

func Name() string {
	return impl.Name()
}

func Wrap(size int, data []float32) Vector {
	return impl.Wrap(size, data)
}

func Dot(a, b Vector) float64 {
	return impl.Dot(a, b)
}
