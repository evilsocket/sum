package main

import (
	"flag"
	"net"
	"os"
	"os/signal"
	"path"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/service"

	"github.com/dustin/go-humanize"
	"github.com/evilsocket/islazy/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

func doCleanup() {
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

func setupSignals() {
	if *cpuProfile != "" {
		if f, err := os.Create(*cpuProfile); err != nil {
			log.Fatal("%v", err)
		} else if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("%v", err)
		}
	}

	sigChan = make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		sig := <-sigChan
		log.Info("got signal %v", sig)
		doCleanup()
		os.Exit(0)
	}()
}

func setupLogging() {
	log.OnFatal = log.ExitOnFatal
	if *logFile != "" {
		log.Output = *logFile
	}

	if *logDebug {
		log.Level = log.DEBUG
	}

	if err := log.Open(); err != nil {
		panic(err)
	}
}

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
	flag.Parse()

	setupSignals()
	setupLogging()
	defer log.Close()

	log.Info("sumd v%s is starting ...", service.Version)

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

	svc, err = service.New(*dataPath, *credsPath, *listenString)
	if err != nil {
		log.Fatal("%v", err)
	}

	grpc.MaxMsgSize(*maxMsgSize)
	server := grpc.NewServer(grpc.Creds(creds))
	pb.RegisterSumServiceServer(server, svc)
	reflection.Register(server)

	go statsReport()

	log.Info("now listening on %s ...", *listenString)
	if err := server.Serve(listener); err != nil {
		log.Fatal("failed to serve: %v", err)
	}
}
