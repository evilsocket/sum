package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/service"

	"github.com/dustin/go-humanize"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	listenString = flag.String("listen", "127.0.0.1:50051", "String to create the TCP listener.")
	dataPath     = flag.String("datapath", "/var/lib/sumd", "Sum data folder.")
	logFile      = flag.String("log-file", "", "If filled, sumd will log to this file.")

	cpuProfile = flag.String("cpu-profile", "", "Write CPU profile to this file.")
	memProfile = flag.String("mem-profile", "", "Write memory profile to this file.")

	svc     = (*service.Service)(nil)
	sigChan = (chan os.Signal)(nil)
)

func doCleanup() {
	log.Printf("shutting down ...")

	if *cpuProfile != "" {
		log.Printf("saving cpu profile to %s ...", *cpuProfile)
		pprof.StopCPUProfile()
	}

	if *memProfile != "" {
		log.Printf("saving memory profile to %s ...", *memProfile)
		f, err := os.Create(*memProfile)
		if err != nil {
			log.Printf("could not create memory profile: %s", err)
			return
		}
		defer f.Close()
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Printf("could not write memory profile: %s", err)
		}
	}
}
func setupSignals() {
	if *cpuProfile != "" {
		if f, err := os.Create(*cpuProfile); err != nil {
			log.Fatal(err)
		} else if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal(err)
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
		log.Printf("got signal %v", sig)
		doCleanup()
		os.Exit(0)
	}()
}

func statsReport() {
	var m runtime.MemStats

	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		runtime.GC()
		runtime.ReadMemStats(&m)

		log.Printf("records:%d oracles:%d mem:%s numgc:%d",
			svc.NumRecords(),
			svc.NumOracles(),
			humanize.Bytes(m.Sys),
			m.NumGC)
	}
}

func main() {
	flag.Parse()

	setupSignals()

	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	listener, err := net.Listen("tcp", *listenString)
	if err != nil {
		log.Fatalf("failed to create listener: %v", err)
	}

	svc, err = service.New(*dataPath)
	if err != nil {
		log.Fatalf("%v", err)
	}

	grpc.MaxMsgSize(10 * 1024 * 1024)
	server := grpc.NewServer()
	pb.RegisterSumServiceServer(server, svc)
	reflection.Register(server)

	go statsReport()

	log.Printf("sumd v%s is listening on %s ...", service.Version, *listenString)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
