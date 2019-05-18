package master

import (
	"context"

	"github.com/evilsocket/sum/node/service"

	. "github.com/evilsocket/sum/proto"
)

// get runtime information about the service
func (ms *Service) Info(ctx context.Context, arg *Empty) (*ServerInfo, error) {
	nRecords := ms.NumRecords()

	ms.idLock.RLock()
	defer ms.idLock.RUnlock()
	ms.cageLock.RLock()
	defer ms.cageLock.RUnlock()

	return service.Info("", ms.credsPath, ms.address, ms.started, nRecords, len(ms.raccoons), ms.nextId), nil
}
