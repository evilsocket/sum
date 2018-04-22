package service

import (
	"errors"
	"fmt"
	"strings"

	pb "github.com/evilsocket/sum/proto"

	"github.com/robertkrimen/otto"
	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/parser"
)

var (
	errNoDeclarations = errors.New("expected function declaration")
)

func validate(oracle *pb.Oracle) (call string, args []string, err error) {
	var prototype *ast.FunctionDeclaration
	// first try to parse the oracle and validate that
	// it starts with a function declaration
	program, err := parser.ParseFile(nil, "", oracle.Code, 0)
	ok := true
	if err != nil {
		return "", nil, err
	} else if program.DeclarationList == nil || len(program.DeclarationList) < 1 {
		return "", nil, errNoDeclarations
	} else if prototype, ok = program.DeclarationList[0].(*ast.FunctionDeclaration); !ok {
		return "", nil, fmt.Errorf("expected function declaration, found %T", program.DeclarationList[0])
	}

	// use the function declaration in order to build  the function call
	args = []string{}
	if prototype.Function.ParameterList != nil && prototype.Function.ParameterList.List != nil {
		args = make([]string, len(prototype.Function.ParameterList.List))
		for i, param := range prototype.Function.ParameterList.List {
			args[i] = param.Name
		}
	}
	call = fmt.Sprintf("%s(%s)",
		prototype.Function.Name.Name,
		strings.Join(args, ","))
	return
}

// Compiles a raw oracle.
func compile(oracle *pb.Oracle) (*compiled, error) {
	callString, args, err := validate(oracle)
	if err != nil {
		return nil, err
	}

	// create the vm and define the oracle function
	vm := otto.New()
	if _, err := vm.Run(oracle.Code); err != nil {
		return nil, err
	}

	// use the vm to precompile the function call
	call, _ := vm.Compile("", callString)

	// done ^_^
	return &compiled{
		oracle: oracle,
		vm:     vm,
		args:   args,
		argc:   len(args),
		call:   call,
	}, nil
}
