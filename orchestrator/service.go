package orchestrator

import (
	"context"
	"os"
	"runtime"
	"time"

	. "github.com/evilsocket/sum/proto"

	"github.com/evilsocket/sum/service"
)

// get runtime information about the service
func (ms *MuxService) Info(ctx context.Context, arg *Empty) (*ServerInfo, error) {
	var m runtime.MemStats

	runtime.ReadMemStats(&m)

	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()
	ms.recordsLock.RLock()
	defer ms.recordsLock.RUnlock()
	ms.cageLock.RLock()
	defer ms.cageLock.RUnlock()

	return &ServerInfo{
		Version:    service.Version,
		Uptime:     uint64(time.Since(ms.started).Seconds()),
		Pid:        ms.pid,
		Uid:        ms.uid,
		Argv:       os.Args,
		Records:    uint64(len(ms.recId2node)),
		Oracles:    uint64(len(ms.raccoons)),
		Os:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		GoVersion:  runtime.Version(),
		Cpus:       uint64(runtime.NumCPU()),
		MaxCpus:    uint64(runtime.GOMAXPROCS(0)),
		Goroutines: uint64(runtime.NumGoroutine()),
		Alloc:      m.Alloc,
		Sys:        m.Sys,
		NumGc:      uint64(m.NumGC),
		Credspath:  ms.credsPath,
		Address:    ms.address,
	}, nil
}
