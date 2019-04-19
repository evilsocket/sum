package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/storage"
	"github.com/evilsocket/sum/wrapper"

	"github.com/robertkrimen/otto"
)

type compiled struct {
	sync.Mutex
	pool   *ExecutionPool
	oracle *pb.Oracle
	call   *otto.Script
	argc   int
	args   []string
}

func (c *compiled) Is(o pb.Oracle) bool {
	return c.oracle.Id == o.Id
}

func dontPanic(err *error) {
	p := recover()
	switch p.(type) {
	case string:
		*err = errors.New(p.(string))
	case error:
		*err = p.(error)
	default:
		*err = fmt.Errorf("got panic of type %T: %v", p, p)
	}
}

func (c *compiled) Run(records *storage.Records, args []string) (ctx *wrapper.Context, raw []byte, err error) {
	ret, err := func() (v otto.Value, err error) {
		defer dontPanic(&err)
		var anyValue interface{}
		// prepare the context that the oracle will be able to use
		// to signal errors and other specific states or events
		ctx = wrapper.NewContext()
		// define the arguments taking into account
		// that some of them might be optional
		for len(args) < c.argc {
			args = append(args, "null")
		}
		// in order to avoid locking the global vm and make this
		// basically single thread, we create a separate clone
		// for each evaluation.
		// NOTE: this will block until a vm is available from the pool.
		vm := c.pool.Get()
		defer vm.Release()
		// define context and globals
		vm.Set("records", wrapper.WrapRecords(records))
		vm.Set("ctx", ctx)
		// define the arguments
		for argIdx := 0; argIdx < c.argc; argIdx++ {
			// unmarshal a typed value from the string value of
			// the argument otherwise vm.Set will define everything
			// as a string
			argRaw := args[argIdx]
			if err = json.Unmarshal([]byte(argRaw), &anyValue); err != nil {
				// NOTE: this error condition is not covered by tests
				// because I couldn't find a way to trigger it giving
				// that the args list is made of simple strings.
				return otto.NullValue(), fmt.Errorf("could not unmarshal value '%s': %s", argRaw, err)
			}
			vm.Set(c.args[argIdx], anyValue)
		}
		// evaluate the function call
		return vm.Run(c.call)
	}()

	if err != nil {
		// do not marshal return value if there's an error
		return ctx, nil, err
	} else if ctx.IsError() {
		// same goes for errors triggered within the oracle
		return ctx, nil, errors.New(ctx.Message())
	} else if obj, err := ret.Export(); err != nil {
		// or if we can't export its return value
		// NOTE: this error condition is not covered by tests
		// because I couldn't find a way to trigger it
		return ctx, nil, err
	} else if raw, err = json.Marshal(obj); err != nil {
		// or if we can't marshal it to a raw buffer for transport
		return ctx, nil, err
	}
	return ctx, raw, nil
}
