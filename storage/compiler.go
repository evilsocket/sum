package storage

import (
	"fmt"
	"strings"

	pb "github.com/evilsocket/sum/proto"

	"github.com/robertkrimen/otto"
)

type CompiledOracle struct {
	vm     *otto.Otto
	oracle *pb.Oracle
}

func Compile(vm *otto.Otto, oracle *pb.Oracle) (*CompiledOracle, error) {
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

func (c *CompiledOracle) Run(args []string) (otto.Value, error) {
	code := fmt.Sprintf("%s(%s)", c.oracle.Name, strings.Join(args, ", "))
	return c.vm.Run(code)
}
