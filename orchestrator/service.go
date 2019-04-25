package main

import (
	"context"
	. "github.com/evilsocket/sum/proto"
	"os"
	"time"
)

func (ms *MuxService) Info(ctx context.Context, arg *Empty) (*ServerInfo, error) {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()
	ms.cageLock.RLock()
	defer ms.cageLock.RUnlock()

	return &ServerInfo{
		Version: Version,
		Uptime:  uint64(time.Since(ms.started).Seconds()),
		Pid:     ms.pid,
		Uid:     ms.uid,
		Argv:    os.Args,
		Records: uint64(len(ms.recId2node)),
		Oracles: uint64(len(ms.raccoons)),
	}, nil
}
