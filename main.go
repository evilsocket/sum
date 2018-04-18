package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	pb "github.com/evilsocket/sum/proto"
	"github.com/satori/go.uuid"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	versionString = "1.0.0b"
	listenString  = ":50051"
)

var (
	service = (*vectorService)(nil)
)

func errorResponse(format string, args ...interface{}) *pb.Response {
	return &pb.Response{Success: false, Msg: fmt.Sprintf(format, args...)}
}

type vectorService struct {
	sync.RWMutex
	started time.Time
	records map[string]*pb.Record
}

func newVectorService() *vectorService {
	return &vectorService{
		started: time.Now(),
		records: make(map[string]*pb.Record),
	}
}

func (s *vectorService) Info(ctx context.Context, dummy *pb.Empty) (*pb.ServerInfo, error) {
	s.RLock()
	defer s.RUnlock()

	return &pb.ServerInfo{
		Version: versionString,
		Uptime:  uint64(time.Since(s.started).Seconds()),
		Pid:     uint64(os.Getpid()),
		Uid:     uint64(os.Getuid()),
		Records: uint64(len(s.records)),
		Argv:    os.Args,
	}, nil
}

func (s *vectorService) Create(ctx context.Context, record *pb.Record) (*pb.Response, error) {
	// if no id was filled, generate a new one
	if record.Id == "" {
		record.Id = uuid.Must(uuid.NewV4()).String()
	}

	s.Lock()
	defer s.Unlock()

	// make sure the id is unique
	if _, found := s.records[record.Id]; found == true {
		return errorResponse("Identifier %s violates the unicity constraint.", record.Id), nil
	}

	s.records[record.Id] = record

	return &pb.Response{Success: true, Msg: record.Id}, nil
}

func (s *vectorService) Update(ctx context.Context, record *pb.Record) (*pb.Response, error) {
	s.Lock()
	defer s.Unlock()

	stored, found := s.records[record.Id]
	if found == false {
		return errorResponse("Record %s not found.", record.Id), nil
	}

	if record.Meta != nil {
		stored.Meta = record.Meta
	}

	if record.Data != nil {
		stored.Data = record.Data
	}

	return &pb.Response{Success: true}, nil
}

func (s *vectorService) Read(ctx context.Context, query *pb.Query) (*pb.Response, error) {
	s.RLock()
	defer s.RUnlock()

	stored, found := s.records[query.Id]
	if found == false {
		return errorResponse("Record %s not found.", query.Id), nil
	}

	return &pb.Response{Success: true, Record: stored}, nil
}

func (s *vectorService) Delete(ctx context.Context, query *pb.Query) (*pb.Response, error) {
	s.Lock()
	defer s.Unlock()

	_, found := s.records[query.Id]
	if found == false {
		return errorResponse("Record %s not found.", query.Id), nil
	}

	delete(s.records, query.Id)

	return &pb.Response{Success: true}, nil
}

func (s *vectorService) numRecords() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.records)
}

func statsReport() {
	var m runtime.MemStats

	ticker := time.NewTicker(5 * time.Second)
	for _ = range ticker.C {
		runtime.ReadMemStats(&m)

		log.Printf("records:%d alloc:%s talloc:%s sys:%s numgc:%d",
			service.numRecords(),
			humanize.Bytes(m.Alloc),
			humanize.Bytes(m.TotalAlloc),
			humanize.Bytes(m.Sys),
			m.NumGC)
	}
}

func main() {
	listener, err := net.Listen("tcp", listenString)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	server := grpc.NewServer()
	service = newVectorService()

	pb.RegisterVectorServiceServer(server, service)

	reflection.Register(server)

	go statsReport()

	log.Printf("listening on %s ...", listenString)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
