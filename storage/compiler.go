package storage

import (
	pb "github.com/evilsocket/sum/proto"

	"github.com/robertkrimen/otto"
)

type CompiledOracle struct {
	oracle *pb.Oracle
}

func Compile(vm *otto.Otto, oracle *pb.Oracle) (*CompiledOracle, error) {
	_, err := vm.Run(oracle.Code)
	if err != nil {
		return nil, err
	}

	return &CompiledOracle{
		oracle: oracle,
	}, nil
}
