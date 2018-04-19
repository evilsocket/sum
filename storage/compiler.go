package storage

import (
	"fmt"
	"strings"
	"sync"

	pb "github.com/evilsocket/sum/proto"

	"github.com/robertkrimen/otto"
)

type CompiledOracle struct {
	sync.Mutex

	vm     *otto.Otto
	oracle *pb.Oracle
}

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

func (c *CompiledOracle) Oracle() *pb.Oracle {
	return c.oracle
}

func (c *CompiledOracle) VM() *otto.Otto {
	return c.vm
}

func (c *CompiledOracle) Run(args []string) (otto.Value, error) {
	c.Lock()
	defer c.Unlock()
	code := fmt.Sprintf("%s(%s)", c.oracle.Name, strings.Join(args, ", "))
	return c.vm.Run(code)
}
