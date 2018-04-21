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
	name   string
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

func (c *compiled) RunWithContext(records *storage.Records, args []string) (*wrapper.Context, error, []byte) {
	var ret otto.Value
	var err error

	ctx := wrapper.NewContext()
	call := fmt.Sprintf("%s(%s)", c.oracle.Name, strings.Join(args, ", "))

	func() {
		c.Lock()
		defer c.Unlock()

		c.vm.Set("records", wrapper.ForRecords(records))
		c.vm.Set("ctx", ctx)

		ret, err = c.vm.Run(call)
	}()

	if err != nil {
		return ctx, err, nil
	} else if ctx.IsError() {
		return ctx, errors.New(ctx.Message()), nil
	}

	obj, _ := ret.Export()
	raw, _ := json.Marshal(obj)

	return ctx, nil, raw
}
