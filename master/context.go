package master

import (
	"context"
	"time"
)

var (
	timeout    = 10 * time.Minute
	maxMsgSize = 50 * 1024 * 1024
)

// create a context to communicate with nodes
func newCommContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

func SetMaxMsgSize(sz int) {
	maxMsgSize = sz
}

func SetCommunicationTimeout(duration time.Duration) {
	timeout = duration
}
