package orchestrator

import (
	"context"
	"time"
)

// update MuxService's nodes periodically
func NodeUpdater(ctx context.Context, ms *MuxService, pollPeriod time.Duration) {
	ticker := time.NewTicker(pollPeriod)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ms.UpdateNodes()
		}
	}
}
