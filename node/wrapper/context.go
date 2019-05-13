package wrapper

import (
	"sync"
)

// Context is a thread safe object passed to oracles during
// execution in order to allow them to signal an error.
type Context struct {
	sync.RWMutex
	message string
	isError bool
}

// NewContext creates a new *Context object.
func NewContext() *Context {
	return &Context{}
}

// Error sets this context to an error state with the given message.
func (ctx *Context) Error(msg string) {
	ctx.Lock()
	defer ctx.Unlock()
	ctx.message = msg
	ctx.isError = true
}

// IsError returns true if an error has been set in this context.
func (ctx *Context) IsError() bool {
	ctx.RLock()
	defer ctx.RUnlock()
	return ctx.isError
}

// Message returns the error message for this context.
func (ctx *Context) Message() string {
	ctx.RLock()
	defer ctx.RUnlock()
	return ctx.message
}

// Reset resets this context instance to a neutral state.
func (ctx *Context) Reset() {
	ctx.Lock()
	defer ctx.Unlock()
	ctx.message = ""
	ctx.isError = false
}
