package main

import (
	"context"
	. "github.com/evilsocket/sum/common"
	pb "github.com/evilsocket/sum/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/reflection"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	nodesStrings = kingpin.Arg("nodes", "nodes to orchestrate").Default("").String()
	timeout      = kingpin.Arg("timeout", "communication timeout").Default("3s").Duration()
	pollPeriod   = kingpin.Arg("pollinterval", "poll interval").Default("500ms").Duration()
	listenString = kingpin.Arg("listen", "String to create the TCP listener.").Default("127.0.0.1:50051").String()
	cpuProfile   = kingpin.Arg("cpu-profile", "Write CPU profile to this file.").Default("").String()
	memProfile   = kingpin.Arg("mem-profile", "Write memory profile to this file.").Default("").String()
	logFile      = kingpin.Arg("log-file", "If filled, sumd will log to this file.").Default("").String()
	logDebug     = kingpin.Arg("debug", "Enable debug logs.").Default("false").Bool()
	maxMsgSize   = kingpin.Arg("max-msg-size", "Maximum size in bytes of a GRPC message.").Default(strconv.Itoa(10 * 1024 * 1024)).Int()
	credsPath    = kingpin.Arg("creds", "Path to the key.pem and cert.pem files to use for TLS based authentication.").Default("/etc/sumd/creds").ExistingDir()
)

const Version = "1.0.0"

// create a context to communicate with nodes
func newCommContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), *timeout)
}

// update MuxService's nodes periodically
func updater(ctx context.Context, ms *MuxService) {
	ticker := time.NewTicker(*pollPeriod)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ms.UpdateNodes()
		}
	}
}

func main() {
	kingpin.Parse()

	StartProfiling(cpuProfile)

	SetupSignals(func(_ os.Signal) { DoCleanup(cpuProfile, memProfile) })

	SetupLogging(logFile, logDebug)
	defer TeardownLogging()

	nodes := make([]*NodeInfo, 0)

	if *nodesStrings != "" {
		for _, n := range strings.Split(*nodesStrings, ",") {
			certFile := filepath.Join(*credsPath, "cert.pem")
			if node, err := createNode(n, certFile); err != nil {
				log.Fatalf("Unable to create node '%s': %v", n, err)
			} else {
				node.ID = uint(len(nodes) + 1)
				nodes = append(nodes, node)
			}
		}
	}

	server, listener := SetupGrpcServer(credsPath, listenString, maxMsgSize)

	ms, err := NewMuxService(nodes)
	if err != nil {
		log.Fatalf("Failed to create MuxService: %v", err)
	}
	ms.credsPath = *credsPath
	ms.address = *listenString

	ctx, cf := context.WithCancel(context.Background())
	defer cf()
	go updater(ctx, ms)

	pb.RegisterSumServiceServer(server, ms)
	pb.RegisterSumMasterServiceServer(server, ms)
	reflection.Register(server)

	log.Printf("sumd-orchestrator v%s is listening on %s ...", Version, *listenString)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
