package backend

// Vector is an opaque interface to whatever the specific backend implementation
// will return as an object wrapper/reference.
type Vector interface{}

// each backend must implement these methods.
type implementation interface {
	Name() string
	Space() uint64

	Wrap(size int, data []float32) Vector

	Dot(a, b Vector) float64
}
