package wrapper

type Vector interface{}

type Backend interface {
	Wrap(size int, data []float32) Vector

	Dot(a, b Vector) float64
}
