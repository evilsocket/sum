package storage

import (
	"fmt"
	"strings"
	"sync"

	pb "github.com/evilsocket/sum/proto"

	"github.com/robertkrimen/otto"
)

// A CompiledOracle is a wrapper for compiled *storage.Oracle
// objects, each one with its dedicated VM.
type CompiledOracle struct {
	sync.Mutex

	vm     *otto.Otto
	oracle *pb.Oracle
}

// Compile takes a raw *pb.Oracle and compiles its code into
// a new *CompiledOracle object.
func Compile(oracle *pb.Oracle) (*CompiledOracle, error) {
	vm := otto.New()

	_, err := vm.Run(oracle.Code)
	if err != nil {
		return nil, err
	}

	return &CompiledOracle{
		vm:     vm,
		oracle: oracle,
	}, nil
}

// Oracle returns the raw *pb.Oracle object.
func (c *CompiledOracle) Oracle() *pb.Oracle {
	return c.oracle
}

// VM returns the dedicate VM object.
func (c *CompiledOracle) VM() *otto.Otto {
	return c.vm
}

// Run locks the VM and evaluates a call to this oracle using the provided
// arguments list.
func (c *CompiledOracle) Run(args []string) (otto.Value, error) {
	c.Lock()
	defer c.Unlock()
	code := fmt.Sprintf("%s(%s)", c.oracle.Name, strings.Join(args, ", "))
	return c.vm.Run(code)
}
