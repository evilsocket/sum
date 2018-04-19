package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/storage"
	"github.com/evilsocket/sum/wrapper"

	"golang.org/x/net/context"
)

type Service struct {
	started time.Time
	pid     uint64
	uid     uint64
	argv    []string
	records *storage.Records
	oracles *storage.Oracles
	ctx     *wrapper.Context
}

func New(dataPath string) (*Service, error) {
	records, err := storage.LoadRecords(filepath.Join(dataPath, "data"))
	if err != nil {
		return nil, err
	}

	oracles, err := storage.LoadOracles(filepath.Join(dataPath, "oracles"))
	if err != nil {
		return nil, err
	}

	vm := oracles.VM()
	ctx := wrapper.NewContext()

	vm.Set("records", wrapper.ForRecords(records))
	vm.Set("ctx", ctx)

	return &Service{
		started: time.Now(),
		pid:     uint64(os.Getpid()),
		uid:     uint64(os.Getuid()),
		argv:    os.Args,
		records: records,
		oracles: oracles,
		ctx:     ctx,
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
		Oracles: s.oracles.Size(),
	}, nil
}

func errCallResponse(format string, args ...interface{}) *pb.CallResponse {
	return &pb.CallResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
}

func (s *Service) Run(ctx context.Context, call *pb.Call) (*pb.CallResponse, error) {
	compiled := s.oracles.Find(call.OracleId)
	if compiled == nil {
		return errCallResponse("Oracle %s not found.", call.OracleId), nil
	}

	// TODO: this should lock the vm and context .. ?
	s.ctx.Reset()

	var data []byte

	if ret, err := compiled.Run(call.Args); err != nil {
		return errCallResponse("Error while running oracle %s: %s", call.OracleId, err), nil
	} else if s.ctx.IsError() {
		return errCallResponse("Error while running oracle %s: %s", call.OracleId, s.ctx.Message()), nil
	} else if obj, err := ret.Export(); err != nil {
		return errCallResponse("Error while serializing return value of oracle %s: %s", call.OracleId, err), nil
	} else if data, err = json.Marshal(obj); err != nil {
		return errCallResponse("Error while marshaling return value of oracle %s: %s", call.OracleId, err), nil
	}

	return &pb.CallResponse{
		Success: true,
		Json:    string(data),
	}, nil
}
