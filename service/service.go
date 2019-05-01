package service

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/storage"

	"github.com/evilsocket/islazy/log"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
)

const (
	// responses bigger than 2K will be gzipped
	gzipResponseSize     = 2048
	gzipCompressionLevel = gzip.BestCompression
	dataFolderName       = "data"
	oraclesFolderName    = "oracles"
)

func errCallResponse(format string, args ...interface{}) *pb.CallResponse {
	return &pb.CallResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
}

// Service represents a single instance of the Sum database
// service.
type Service struct {
	sync.RWMutex

	datapath  string
	credspath string
	address   string
	started   time.Time
	pid       uint64
	uid       uint64
	argv      []string
	records   *storage.Records
	oracles   *storage.Oracles
	cache     *compiledCache
}

// New loads records and oracles from a given path and returns
// a new instance of the *Service object.
func New(dataPath string, credsPath string, address string) (svc *Service, err error) {
	if dataPath, err = filepath.Abs(dataPath); err != nil {
		return nil, err
	} else if credsPath, err = filepath.Abs(credsPath); err != nil {
		return nil, err
	}

	records, err := storage.LoadRecords(filepath.Join(dataPath, dataFolderName))
	if err != nil {
		return nil, err
	}

	oracles, err := storage.LoadOracles(filepath.Join(dataPath, oraclesFolderName))
	if err != nil {
		return nil, err
	}

	svc = &Service{
		datapath:  dataPath,
		credspath: credsPath,
		address:   address,
		started:   time.Now(),
		pid:       uint64(os.Getpid()),
		uid:       uint64(os.Getuid()),
		argv:      os.Args,
		records:   records,
		oracles:   oracles,
		cache:     newCache(),
	}

	if oracles.Size() > 0 {
		log.Info("precompiling %d oracles ...", oracles.Size())
		err := oracles.ForEach(func(m proto.Message) error {
			oracle := m.(*pb.Oracle)
			compiled, err := compile(oracle)
			if err != nil {
				return fmt.Errorf("error while compiling oracle %d: %s", oracle.Id, err)
			}
			svc.cache.Add(oracle.Id, compiled)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return svc, nil
}

// Info returns a *pb.ServerInfo object with various realtime information
// about the service and its runtime.
func (s *Service) Info(ctx context.Context, dummy *pb.Empty) (*pb.ServerInfo, error) {
	var m runtime.MemStats

	runtime.ReadMemStats(&m)

	return &pb.ServerInfo{
		Version:    Version,
		Os:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		GoVersion:  runtime.Version(),
		Cpus:       uint64(runtime.NumCPU()),
		MaxCpus:    uint64(runtime.GOMAXPROCS(0)),
		Goroutines: uint64(runtime.NumGoroutine()),
		Alloc:      m.Alloc,
		Sys:        m.Sys,
		NumGc:      uint64(m.NumGC),
		Datapath:   s.datapath,
		Credspath:  s.credspath,
		Address:    s.address,
		Uptime:     uint64(time.Since(s.started).Seconds()),
		Pid:        s.pid,
		Uid:        s.uid,
		Argv:       s.argv,
		Records:    uint64(s.records.Size()),
		Oracles:    uint64(s.oracles.Size()),
	}, nil
}

func buildPayload(raw []byte) *pb.Data {
	data := pb.Data{
		Compressed: false,
		Payload:    raw,
	}
	// compress the payload if needed
	if len(raw) > gzipResponseSize {
		var buf bytes.Buffer
		if compress, err := gzip.NewWriterLevel(&buf, gzipCompressionLevel); err == nil {
			wrote, err := compress.Write(raw)
			compress.Close()
			if wrote > 0 && err == nil {
				data.Compressed = true
				data.Payload = buf.Bytes()
			}
		}
	}
	return &data
}

// Run executes a compiled oracle given its identifier and the arguments
// in the *pb.Call object.
func (s *Service) Run(ctx context.Context, call *pb.Call) (resp *pb.CallResponse, err error) {
	compiled := s.cache.Get(call.OracleId)
	if compiled == nil {
		return errCallResponse("oracle %d not found.", call.OracleId), nil
	}

	defer func() {
		if v := recover(); v != nil {
			err = v.(error)
			log.Debug("%s", err)
			resp = errCallResponse("error while running oracle %d: %s", call.OracleId, err)
		}
	}()

	log.Debug("call: %+v", call)

	_, raw, err := compiled.Run(s.records, call.Args)
	if err != nil {
		return errCallResponse("error while running oracle %d: %s", call.OracleId, err), nil
	}

	resp = &pb.CallResponse{
		Success: true,
		Data:    buildPayload(raw),
	}
	return resp, err
}
