package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"
	"path"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/evilsocket/sum/master"
	node "github.com/evilsocket/sum/node/service"

	pb "github.com/evilsocket/sum/proto"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	"github.com/dustin/go-humanize"
	"github.com/evilsocket/islazy/log"
)

var (
	// node/slave
	listenString = flag.String("listen", "127.0.0.1:50051", "String to create the TCP listener.")
	credsPath    = flag.String("creds", "/etc/sumd/creds", "Path to the key.pem and cert.pem files to use for TLS based authentication.")
	dataPath     = flag.String("datapath", "/var/lib/sumd", "Sum data folder.")
	gcPeriod     = flag.Int("gc-period", 1800, "Period in seconds to report memory statistics and call the gc.")
	maxMsgSize   = flag.Int("max-msg-size", 10*1024*1024, "Maximum size in bytes of a GRPC message.")
	logFile      = flag.String("log-file", "", "If filled, sumd will log to this file.")
	logDebug     = flag.Bool("debug", false, "Enable debug logs.")

	// master
	masterCfgFile = flag.String("master", "", "Load sum master configuration and become the master.")
	timeout       = flag.Duration("timeout", 10*time.Minute, "nodes communication timeout")
	pollPeriod    = flag.Duration("poll", 1*time.Second, "nodes poll interval")

	// profiling
	cpuProfile = flag.String("cpu-profile", "", "Write CPU profile to this file.")
	memProfile = flag.String("mem-profile", "", "Write memory profile to this file.")

	nodeSvc   = (*node.Service)(nil)
	masterSvc = (*master.Service)(nil)
)

func statsReport() {
	var m runtime.MemStats
	var reporter interface {
		NumRecords() int
		NumOracles() int
	}

	if nodeSvc != nil {
		reporter = nodeSvc
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

	startProfiling(cpuProfile)

	setupSignals(func(_ os.Signal) { doCleanup(cpuProfile, memProfile) })

	setupLogging(logFile, logDebug)
	defer teardownLogging()

	mode := "node/slave"
	if *masterCfgFile != "" {
		mode = "master"
	}

	log.Info("sumd v%s is starting as %s on %s (%s) ...", node.Version, mode, *listenString, *credsPath)

	server, listener := setupGrpcServer(credsPath, listenString, maxMsgSize)

	if *masterCfgFile != "" {
		master.SetCommunicationTimeout(*timeout)

		if masterSvc, err = master.NewServiceFromConfig(*masterCfgFile, *credsPath, *listenString); err != nil {
			log.Fatal("cannot start master service: %v", err)
		}

		pb.RegisterSumMasterServiceServer(server, masterSvc)
		pb.RegisterSumInternalServiceServer(server, masterSvc)
		pb.RegisterSumServiceServer(server, masterSvc)

		ctx, cf := context.WithCancel(context.Background())
		defer cf()

		go master.NodeUpdater(ctx, masterSvc, *pollPeriod)
	} else {
		if nodeSvc, err = node.New(*dataPath, *credsPath, *listenString); err != nil {
			log.Fatal("%v", err)
		}
		pb.RegisterSumInternalServiceServer(server, nodeSvc)
		pb.RegisterSumServiceServer(server, nodeSvc)
	}

	go statsReport()

	reflection.Register(server)

	log.Info("now listening on %s ...", *listenString)
	if err := server.Serve(listener); err != nil {
		log.Fatal("failed to serve: %v", err)
	}
}

func startProfiling(cpuProfile *string) {
	if *cpuProfile == "" {
		return
	}

	if f, err := os.Create(*cpuProfile); err != nil {
		log.Fatal("%v", err)
	} else if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("%v", err)
	}
}

func setupSignals(handlers ...func(os.Signal)) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		sig := <-sigChan
		log.Info("got signal %v", sig)
		for _, handler := range handlers {
			handler(sig)
		}

		os.Exit(0)
	}()
}

func doCleanup(cpuProfile, memProfile *string) {
	log.Info("shutting down ...")

	if *cpuProfile != "" {
		log.Info("saving cpu profile to %s ...", *cpuProfile)
		pprof.StopCPUProfile()
	}

	if *memProfile != "" {
		log.Info("saving memory profile to %s ...", *memProfile)
		f, err := os.Create(*memProfile)
		if err != nil {
			log.Info("could not create memory profile: %s", err)
			return
		}
		defer f.Close()
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Info("could not write memory profile: %s", err)
		}
	}
}

func setupLogging(logFile *string, logDebug *bool) {
	log.OnFatal = log.ExitOnFatal
	if *logFile != "" {
		log.Output = *logFile

		f, err := os.Open(*logFile)
		if err != nil {
			panic(err)
		}

		logrus.SetOutput(f)
	}

	if *logDebug {
		log.Level = log.DEBUG
		logrus.SetLevel(logrus.DebugLevel)
	}

	if err := log.Open(); err != nil {
		panic(err)
	}
}

func teardownLogging() {
	log.Close()
}

func setupGrpcServer(credsPath, listenString *string, maxMsgSize *int) (*grpc.Server, net.Listener) {
	crtFile := path.Join(*credsPath, "cert.pem")
	keyFile := path.Join(*credsPath, "key.pem")
	creds, err := credentials.NewServerTLSFromFile(crtFile, keyFile)
	if err != nil {
		log.Fatal("failed to load credentials from %s: %v", *credsPath, err)
	}

	listener, err := net.Listen("tcp", *listenString)
	if err != nil {
		log.Fatal("failed to create listener: %v", err)
	}

	grpc.MaxRecvMsgSize(*maxMsgSize)
	grpc.MaxSendMsgSize(*maxMsgSize)

	server := grpc.NewServer(grpc.Creds(creds))

	return server, listener
}
