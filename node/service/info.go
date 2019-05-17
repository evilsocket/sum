package service

import (
	"os"
	"runtime"
	"time"

	"github.com/evilsocket/sum/node/backend"

	pb "github.com/evilsocket/sum/proto"
)

// Info returns a *pb.ServerInfo object with various realtime information
// about the service and its runtime.
func Info(datapath, credspath, address string, started time.Time, records, oracles int) *pb.ServerInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &pb.ServerInfo{
		Version:      Version,
		Os:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		GoVersion:    runtime.Version(),
		Cpus:         uint64(runtime.NumCPU()),
		MaxCpus:      uint64(runtime.GOMAXPROCS(0)),
		Goroutines:   uint64(runtime.NumGoroutine()),
		Alloc:        m.Alloc,
		Sys:          m.Sys,
		NumGc:        uint64(m.NumGC),
		Uptime:       uint64(time.Since(started).Seconds()),
		Records:      uint64(records),
		Oracles:      uint64(oracles),
		Backend:      backend.Name(),
		BackendSpace: backend.Space(),
		Pid:          uint64(os.Getpid()),
		Uid:          uint64(os.Getuid()),
		Argv:         os.Args,
		Datapath:     datapath,
		Credspath:    credspath,
		Address:      address,
	}
}
