package orchestrator

import (
	"context"
	"time"
)

const Version = "1.0.0"

var timeout = 3 * time.Second

// create a context to communicate with nodes
func newCommContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

func SetCommunicationTimeout(duration time.Duration) {
	timeout = duration
}
