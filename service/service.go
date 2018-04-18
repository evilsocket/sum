package service

import (
	"fmt"
	"os"
	"time"

	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/storage"

	"golang.org/x/net/context"
)

type Service struct {
	started time.Time
	pid     uint64
	uid     uint64
	argv    []string
	storage *storage.Storage
}

func errorResponse(format string, args ...interface{}) *pb.Response {
	return &pb.Response{Success: false, Msg: fmt.Sprintf(format, args...)}
}

func New(dataPath string) (*Service, error) {
	storage, err := storage.New(dataPath)
	if err != nil {
		return nil, err
	}
	return &Service{
		started: time.Now(),
		pid:     uint64(os.Getpid()),
		uid:     uint64(os.Getuid()),
		argv:    os.Args,
		storage: storage,
	}, nil
}

func (s *Service) StorageSize() uint64 {
	return s.storage.Size()
}

func (s *Service) Info(ctx context.Context, dummy *pb.Empty) (*pb.ServerInfo, error) {
	return &pb.ServerInfo{
		Version: Version,
		Uptime:  uint64(time.Since(s.started).Seconds()),
		Pid:     s.pid,
		Uid:     s.uid,
		Argv:    s.argv,
		Records: s.storage.Size(),
	}, nil
}

func (s *Service) Create(ctx context.Context, record *pb.Record) (*pb.Response, error) {
	if err := s.storage.Create(record); err != nil {
		return errorResponse("%s", err), nil
	}
	return &pb.Response{Success: true, Msg: record.Id}, nil
}

func (s *Service) Update(ctx context.Context, record *pb.Record) (*pb.Response, error) {
	if err := s.storage.Update(record); err != nil {
		return errorResponse("%s", err), nil
	}
	return &pb.Response{Success: true}, nil
}

func (s *Service) Read(ctx context.Context, query *pb.Query) (*pb.Response, error) {
	record := s.storage.Find(query.Id)
	if record == nil {
		return errorResponse("Record %s not found.", query.Id), nil
	}
	return &pb.Response{Success: true, Record: record}, nil
}

func (s *Service) Delete(ctx context.Context, query *pb.Query) (*pb.Response, error) {
	record := s.storage.Delete(query.Id)
	if record == nil {
		return errorResponse("Record %s not found.", query.Id), nil
	}
	return &pb.Response{Success: true}, nil
}
