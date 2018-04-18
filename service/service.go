package service

import (
	"fmt"
	"os"
	"path/filepath"
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
	records *storage.Records
}

func errorResponse(format string, args ...interface{}) *pb.Response {
	return &pb.Response{Success: false, Msg: fmt.Sprintf(format, args...)}
}

func New(dataPath string) (*Service, error) {
	records, err := storage.LoadRecords(filepath.Join(dataPath, "data"))
	if err != nil {
		return nil, err
	}
	return &Service{
		started: time.Now(),
		pid:     uint64(os.Getpid()),
		uid:     uint64(os.Getuid()),
		argv:    os.Args,
		records: records,
	}, nil
}

func (s *Service) NumRecords() uint64 {
	return s.records.Size()
}

func (s *Service) Info(ctx context.Context, dummy *pb.Empty) (*pb.ServerInfo, error) {
	return &pb.ServerInfo{
		Version: Version,
		Uptime:  uint64(time.Since(s.started).Seconds()),
		Pid:     s.pid,
		Uid:     s.uid,
		Argv:    s.argv,
		Records: s.records.Size(),
	}, nil
}

func (s *Service) Create(ctx context.Context, record *pb.Record) (*pb.Response, error) {
	if err := s.records.Create(record); err != nil {
		return errorResponse("%s", err), nil
	}
	return &pb.Response{Success: true, Msg: record.Id}, nil
}

func (s *Service) Update(ctx context.Context, record *pb.Record) (*pb.Response, error) {
	if err := s.records.Update(record); err != nil {
		return errorResponse("%s", err), nil
	}
	return &pb.Response{Success: true}, nil
}

func (s *Service) Read(ctx context.Context, query *pb.Query) (*pb.Response, error) {
	record := s.records.Find(query.Id)
	if record == nil {
		return errorResponse("Record %s not found.", query.Id), nil
	}
	return &pb.Response{Success: true, Record: record}, nil
}

func (s *Service) Delete(ctx context.Context, query *pb.Query) (*pb.Response, error) {
	record := s.records.Delete(query.Id)
	if record == nil {
		return errorResponse("Record %s not found.", query.Id), nil
	}
	return &pb.Response{Success: true}, nil
}
