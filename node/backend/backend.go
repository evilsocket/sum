package backend

// Vector is an opaque interface to whatever the specific backend implementation
// will return as an object wrapper/reference.
type Vector interface{}

type implementation interface {
	Name() string

	Wrap(size int, data []float32) Vector

	Dot(a, b Vector) float64
}

// TODO: pick at runtime the best backend available ( CUDA, OpenCL or BLAS32 ).
var impl = blas{}

// Name returns the name of the current backend.
func Name() string {
	return impl.Name()
}

// Wrap creates an opaque reference to the data, backend specific.
func Wrap(size int, data []float32) Vector {
	return impl.Wrap(size, data)
}

// Dot performs the dot product between a vector and another.
func Dot(a, b Vector) float64 {
	return impl.Dot(a, b)
}
