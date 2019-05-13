package orchestrator

import (
	"context"
	"time"
)

var timeout = 10 * time.Minute

// create a context to communicate with nodes
func newCommContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

func SetCommunicationTimeout(duration time.Duration) {
	timeout = duration
}
