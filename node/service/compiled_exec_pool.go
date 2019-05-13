package service

import (
	"github.com/robertkrimen/otto"
)

const (
	numVMInPool  = 8
	busyVMMarker = -1
)

// VM is only used to wrap the vm object and give
// a Release method the caller can defer in
// order to free the VM itself transparently.
type VM struct {
	*otto.Otto
	parent *ExecutionPool
	index  int
}

// Release adds this object back to the free list of the pool.
func (w *VM) Release() {
	w.parent.setFree(w.index)
}

// ExecutionPool is a pool of clones of a single VM that
// will be used to scale the execution of an oracle to different
// goroutines without locking one single shared VM.
type ExecutionPool struct {
	root     *otto.Otto
	clones   []*VM
	freeList []int
	freeWay  chan int
}

// CreateExecutionPool creates an ExecutionPool object for the given VM.
func CreateExecutionPool(vm *otto.Otto) *ExecutionPool {
	p := &ExecutionPool{
		root:     vm,
		clones:   make([]*VM, numVMInPool),
		freeList: make([]int, numVMInPool),
		freeWay:  make(chan int, numVMInPool),
	}

	for i := 0; i < numVMInPool; i++ {
		p.freeList[i] = i
		p.clones[i] = &VM{
			Otto:   p.root.Copy(),
			parent: p,
			index:  i,
		}
		// presignal a free object so the very first
		// loop won't wait
		p.freeWay <- i
	}

	return p
}

func (p *ExecutionPool) setFree(index int) {
	p.freeWay <- index
	p.freeList[index] = busyVMMarker
}

// Get will wait until a VM object in the pool is signaled
// as free by whoever was using it and then return the first
// signaled free VM instance.
func (p *ExecutionPool) Get() *VM {
	for freeIndex := range p.freeWay {
		return p.clones[freeIndex]
	}

	return nil
}
