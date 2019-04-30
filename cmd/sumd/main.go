package main

import (
	"flag"
	. "github.com/evilsocket/sum/common"
	"os"
	"runtime"
	"time"

	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/service"

	"github.com/dustin/go-humanize"
	"github.com/evilsocket/islazy/log"
	"google.golang.org/grpc/reflection"
)

var (
	listenString = flag.String("listen", "127.0.0.1:50051", "String to create the TCP listener.")
	credsPath    = flag.String("creds", "/etc/sumd/creds", "Path to the key.pem and cert.pem files to use for TLS based authentication.")
	dataPath     = flag.String("datapath", "/var/lib/sumd", "Sum data folder.")
	gcPeriod     = flag.Int("gc-period", 1800, "Period in seconds to report memory statistics and call the gc.")
	maxMsgSize   = flag.Int("max-msg-size", 10*1024*1024, "Maximum size in bytes of a GRPC message.")
	logFile      = flag.String("log-file", "", "If filled, sumd will log to this file.")
	logDebug     = flag.Bool("debug", false, "Enable debug logs.")

	cpuProfile = flag.String("cpu-profile", "", "Write CPU profile to this file.")
	memProfile = flag.String("mem-profile", "", "Write memory profile to this file.")

	svc     = (*service.Service)(nil)
	sigChan = (chan os.Signal)(nil)
)

func statsReport() {
	var m runtime.MemStats

	ticker := time.NewTicker(time.Duration(*gcPeriod) * time.Second)
	for range ticker.C {
		runtime.GC()
		runtime.ReadMemStats(&m)

		log.Info("records:%d oracles:%d mem:%s numgc:%d",
			svc.NumRecords(),
			svc.NumOracles(),
			humanize.Bytes(m.Sys),
			m.NumGC)
	}
}

func main() {
	var err error

	flag.Parse()

	StartProfiling(cpuProfile)

	SetupSignals(func(_ os.Signal) { DoCleanup(cpuProfile, memProfile) })

	SetupLogging(logFile, logDebug)
	defer TeardownLogging()

	log.Info("sumd v%s is starting ...", service.Version)

	svc, err = service.New(*dataPath, *credsPath, *listenString)
	if err != nil {
		log.Fatal("%v", err)
	}

	server, listener := SetupGrpcServer(credsPath, listenString, maxMsgSize)

	pb.RegisterSumServiceServer(server, svc)
	pb.RegisterSumInternalServiceServer(server, svc)
	reflection.Register(server)

	go statsReport()

	log.Info("now listening on %s ...", *listenString)
	if err := server.Serve(listener); err != nil {
		log.Fatal("failed to serve: %v", err)
	}
}
