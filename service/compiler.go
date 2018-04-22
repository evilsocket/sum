package service

import (
	pb "github.com/evilsocket/sum/proto"

	"github.com/robertkrimen/otto"
)

// Compiles a raw oracle.
func compile(oracle *pb.Oracle) (*compiled, error) {
	// create the vm and define the oracle function
	vm := otto.New()
	if _, err := vm.Run(oracle.Code); err != nil {
		return nil, err
	}

	// TODO: validate args

	return &compiled{
		oracle: oracle,
		vm:     vm,
	}, nil
}
