package main

import (
	"context"
	"flag"
	. "github.com/evilsocket/sum/common"
	"github.com/evilsocket/sum/orchestrator"
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

	// orchestrator

	masterCfgFile = flag.String("master-cfg", "", "Load sum master configuration and become the master.")
	timeout       = flag.Duration("timeout", 3*time.Second, "nodes communication timeout")
	pollPeriod    = flag.Duration("pollinterval", 500*time.Millisecond, "nodes poll interval")

	// stats

	cpuProfile = flag.String("cpu-profile", "", "Write CPU profile to this file.")
	memProfile = flag.String("mem-profile", "", "Write memory profile to this file.")

	svc       = (*service.Service)(nil)
	masterSvc = (*orchestrator.MuxService)(nil)
)

func statsReport() {
	var m runtime.MemStats
	var reporter interface {
		NumRecords() int
		NumOracles() int
	}

	if svc != nil {
		reporter = svc
	} else if masterSvc != nil {
		reporter = masterSvc
	} else {
		panic("no service has been created")
	}

	ticker := time.NewTicker(time.Duration(*gcPeriod) * time.Second)
	for range ticker.C {
		runtime.GC()
		runtime.ReadMemStats(&m)

		log.Info("records:%d oracles:%d mem:%s numgc:%d",
			reporter.NumRecords(),
			reporter.NumOracles(),
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

	server, listener := SetupGrpcServer(credsPath, listenString, maxMsgSize)

	if *masterCfgFile != "" {

		orchestrator.SetCommunicationTimeout(*timeout)

		if masterSvc, err = orchestrator.NewMuxServiceFromConfig(*masterCfgFile, *credsPath, *listenString); err != nil {
			log.Fatal("Cannot start master service: %v", err)
		}

		pb.RegisterSumMasterServiceServer(server, masterSvc)
		pb.RegisterSumServiceServer(server, masterSvc)

		ctx, cf := context.WithCancel(context.Background())
		defer cf()

		go orchestrator.NodeUpdater(ctx, masterSvc, *pollPeriod)
	} else {
		if svc, err = service.New(*dataPath, *credsPath, *listenString); err != nil {
			log.Fatal("%v", err)
		}
		pb.RegisterSumInternalServiceServer(server, svc)
		pb.RegisterSumServiceServer(server, svc)
	}

	go statsReport()

	reflection.Register(server)

	log.Info("now listening on %s ...", *listenString)
	if err := server.Serve(listener); err != nil {
		log.Fatal("failed to serve: %v", err)
	}
}
