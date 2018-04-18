package service

import (
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
