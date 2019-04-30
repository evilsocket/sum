package main

import (
	"context"
	pb "github.com/evilsocket/sum/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gopkg.in/alecthomas/kingpin.v2"
	"net"
	"strings"
	"time"
)

var (
	nodesStrings = kingpin.Arg("nodes", "nodes to orchestrate").Required().String()
	timeout      = kingpin.Arg("timeout", "communication timeout").Default("3s").Duration()
	pollPeriod   = kingpin.Arg("pollinterval", "poll interval").Default("500ms").Duration()
	listenString = kingpin.Arg("listen", "String to create the TCP listener.").Default("127.0.0.1:50051").String()
)

const Version = "1.0.0"

func newCommContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), *timeout)
}

func updater(ctx context.Context, ms *MuxService) {
	ticker := time.NewTicker(*pollPeriod)

	select {
	case <-ctx.Done():
		return
	case <-ticker.C:
		ms.UpdateNodes()
	}
}

func main() {
	kingpin.Parse()

	nodes := make([]*NodeInfo, 0)

	for _, n := range strings.Split(*nodesStrings, ",") {
		if node, err := createNode(n); err != nil {
			log.Fatalf("Unable to create node '%s': %v", n, err)
		} else {
			node.ID = uint(len(nodes) + 1)
			nodes = append(nodes, node)
		}
	}

	listener, err := net.Listen("tcp", *listenString)
	if err != nil {
		log.Fatalf("failed to create listener: %v", err)
	}

	ms, err := NewMuxService(nodes)
	if err != nil {
		log.Fatalf("Failed to create MuxService: %v", err)
	}

	ctx, cf := context.WithCancel(context.Background())
	defer cf()
	go updater(ctx, ms)

	server := grpc.NewServer()
	pb.RegisterSumServiceServer(server, ms)
	reflection.Register(server)

	log.Printf("sumd-orchestrator v%s is listening on %s ...", Version, *listenString)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
