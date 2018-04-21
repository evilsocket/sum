package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/storage"
	"github.com/evilsocket/sum/wrapper"

	"github.com/robertkrimen/otto"
)

type compiled struct {
	sync.Mutex
	vm     *otto.Otto
	oracle *pb.Oracle
}

func compileOracle(oracle *pb.Oracle) (*compiled, error) {
	vm := otto.New()
	_, err := vm.Run(oracle.Code)
	if err != nil {
		return nil, err
	}

	return &compiled{
		oracle: oracle,
		vm:     vm,
	}, nil
}

func (c *compiled) Oracle() *pb.Oracle {
	return c.oracle
}

func (c *compiled) RunWithContext(records *storage.Records, args []string) (*wrapper.Context, []byte, error) {
	var ret otto.Value
	var err error

	// FIXME: this should be built as a small AST tree in order to
	// avoid parsing.
	call := fmt.Sprintf("%s(%s)", c.oracle.Name, strings.Join(args, ", "))
	ctx := wrapper.NewContext()
	func() {
		c.Lock()
		defer c.Unlock()

		c.vm.Set("records", wrapper.ForRecords(records))
		c.vm.Set("ctx", ctx)

		ret, err = c.vm.Run(call)
	}()

	if err != nil {
		return ctx, nil, err
	} else if ctx.IsError() {
		return ctx, nil, errors.New(ctx.Message())
	}

	obj, _ := ret.Export()
	raw, _ := json.Marshal(obj)

	return ctx, raw, nil
}
