package common

import (
	"github.com/evilsocket/islazy/log"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net"
	"os"
	"os/signal"
	"path"
	"runtime"
	"runtime/pprof"
	"syscall"
)

func StartProfiling(cpuProfile *string) {
	if *cpuProfile == "" {
		return
	}

	if f, err := os.Create(*cpuProfile); err != nil {
		log.Fatal("%v", err)
	} else if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("%v", err)
	}
}

func SetupSignals(handlers ...func(os.Signal)) {
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

func DoCleanup(cpuProfile, memProfile *string) {
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

func SetupLogging(logFile *string, logDebug *bool) {
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

func TeardownLogging() {
	log.Close()
}

func SetupGrpcServer(credsPath, listenString *string, maxMsgSize *int) (*grpc.Server, net.Listener) {
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

	grpc.MaxMsgSize(*maxMsgSize)
	server := grpc.NewServer(grpc.Creds(creds))

	return server, listener
}
