package wrapper

type Vector interface{}

type Backend interface {
	Wrap(size int, data []float64) Vector
	Equal(a, b Vector) bool
	EqualApprox(a, b Vector, epsilon float64) bool
	Max(a Vector) float64
	Min(a Vector) float64
	Sum(a Vector) float64
	Dot(a, b Vector) float64
	Det(a Vector) float64
}
