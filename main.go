package main

import (
	"flag"
	"log"
	"net"
	"runtime"
	"time"

	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/service"

	"github.com/dustin/go-humanize"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	listenString = flag.String("listen", ":50051", "String to create the TCP listener.")
	dataPath     = flag.String("datapath", "/var/lib/sumd", "Sum data folder.")

	svc = (*service.Service)(nil)
)

func statsReport() {
	var m runtime.MemStats

	ticker := time.NewTicker(5 * time.Second)
	for _ = range ticker.C {
		runtime.GC()
		runtime.ReadMemStats(&m)

		log.Printf("records:%d mem:%s numgc:%d",
			svc.NumRecords(),
			humanize.Bytes(m.Sys),
			m.NumGC)
	}
}

func main() {
	flag.Parse()

	listener, err := net.Listen("tcp", *listenString)
	if err != nil {
		log.Fatalf("failed to create listener: %v", err)
	}

	svc, err = service.New(*dataPath)
	if err != nil {
		log.Fatalf("%v", err)
	}

	server := grpc.NewServer()
	pb.RegisterSumServiceServer(server, svc)
	reflection.Register(server)

	go statsReport()

	log.Printf("sumd v%s is listening on %s ...", service.Version, *listenString)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
