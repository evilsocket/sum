package backend

type naive struct {
}

func (impl naive) Name() string {
	return "naive"
}

func (impl naive) Wrap(size int, data []float32) Vector {
	return data
}

func (impl naive) Dot(a, b Vector) float64 {
	dot := float64(0.0)
	for i, va := range a.([]float32) {
		vb := b.([]float32)[i]
		dot += float64(va) * float64(vb)
	}
	return dot
}
