package service

import (
	"encoding/json"
	"errors"
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
	call   *otto.Script
	args   []string
}

func (c *compiled) Oracle() *pb.Oracle {
	return c.oracle
}

func (c *compiled) RunWithContext(records *storage.Records, args []string) (*wrapper.Context, []byte, error) {
	var ret otto.Value
	var err error

	ctx := wrapper.NewContext()
	func() {
		c.Lock()
		defer c.Unlock()

		// define context and globals
		c.vm.Set("records", wrapper.WrapRecords(records))
		c.vm.Set("ctx", ctx)

		// define the arguments
		for argIdx, argName := range args {
			c.vm.Set(argName, args[argIdx])
		}

		// evaluate the precompiled function call
		ret, err = c.vm.Run(c.call)
	}()

	if err != nil {
		return ctx, nil, err
	} else if ctx.IsError() {
		return ctx, nil, errors.New(ctx.Message())
	}

	// TODO: find a more efficient way to transparently
	// encode oracles return values as I suspect this is
	// not the optimal approach ... ?
	obj, _ := ret.Export()
	raw, _ := json.Marshal(obj)

	return ctx, raw, nil
}
