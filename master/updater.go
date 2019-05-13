package master

import (
	"context"
	"time"
)

// update Service's nodes periodically
func NodeUpdater(ctx context.Context, ms *Service, pollPeriod time.Duration) {
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
