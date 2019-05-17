package backend

import (
	"fmt"
	"sync"
)

var lock = sync.Mutex{}

var available = map[string]implementation{
	"naive":  naive{},
	"blas32": blas{},
}

// TODO: pick at runtime the best backend available ( CUDA, OpenCL or BLAS32 ).
var selected = available["blas32"]

func Available() []string {
	keys := []string{}
	for k, _ := range available {
		keys = append(keys, k)
	}
	return keys
}

// Select selects the current backend by name.
func Select(name string) {
	lock.Lock()
	defer lock.Unlock()

	if impl, found := available[name]; found {
		selected = impl
	} else {
		panic(fmt.Errorf("backend %s is not available", name))
	}
}

// Name returns the name of the current backend.
func Name() string {
	lock.Lock()
	defer lock.Unlock()
	return selected.Name()
}

// Space returns the total available memory in bytes to the specific backend.
func Space() uint64 {
	lock.Lock()
	defer lock.Unlock()
	return selected.Space()
}

// Wrap creates an opaque reference to the data, backend specific.
func Wrap(size int, data []float32) Vector {
	lock.Lock()
	defer lock.Unlock()
	return selected.Wrap(size, data)
}

// Dot performs the dot product between a vector and another.
func Dot(a, b Vector) float64 {
	lock.Lock()
	defer lock.Unlock()
	return selected.Dot(a, b)
}
