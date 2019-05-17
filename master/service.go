package master

import (
	"context"

	"github.com/evilsocket/sum/node/service"

	. "github.com/evilsocket/sum/proto"
)

// get runtime information about the service
func (ms *Service) Info(ctx context.Context, arg *Empty) (*ServerInfo, error) {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()
	ms.idLock.RLock()
	defer ms.idLock.RUnlock()
	ms.cageLock.RLock()
	defer ms.cageLock.RUnlock()

	return service.Info("", ms.credsPath, ms.address, ms.started, ms.NumRecords(), len(ms.raccoons), ms.nextId), nil
}
