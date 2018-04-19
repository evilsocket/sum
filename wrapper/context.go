package wrapper

import (
	"sync"
)

type Context struct {
	sync.RWMutex
	message string
	isError bool
}

func NewContext() *Context {
	return &Context{}
}

func (ctx *Context) Reset() {
	ctx.Lock()
	defer ctx.Unlock()
	ctx.message = ""
	ctx.isError = false
}

func (ctx *Context) Error(msg string) {
	ctx.Lock()
	defer ctx.Unlock()
	ctx.message = msg
	ctx.isError = true
}

func (ctx *Context) IsError() bool {
	ctx.RLock()
	defer ctx.RUnlock()
	return ctx.isError
}

func (ctx *Context) Message() string {
	ctx.RLock()
	defer ctx.RUnlock()
	return ctx.message
}
