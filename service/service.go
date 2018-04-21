package service

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/storage"

	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
)

const (
	// responses bigger than 2K will be gzipped
	gzipResponseSize  = 2048
	dataFolderName    = "data"
	oraclesFolderName = "oracles"
)

func errCallResponse(format string, args ...interface{}) *pb.CallResponse {
	return &pb.CallResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
}

// Service represents a single instance of the Sum database
// service.
type Service struct {
	sync.RWMutex

	started time.Time
	pid     uint64
	uid     uint64
	argv    []string
	records *storage.Records
	oracles *storage.Oracles
	cache   *compiledCache
}

// New loads records and oracles from a given path and returns
// a new instance of the *Service object.
func New(dataPath string) (*Service, error) {
	records, err := storage.LoadRecords(filepath.Join(dataPath, dataFolderName))
	if err != nil {
		return nil, err
	}

	oracles, err := storage.LoadOracles(filepath.Join(dataPath, oraclesFolderName))
	if err != nil {
		return nil, err
	}

	cache := newCache()
	if oracles.Size() > 0 {
		log.Printf("precompiling %d oracles ...", oracles.Size())
		err := error(nil)
		c := (*compiled)(nil)
		oracles.ForEach(func(m proto.Message) {
			if err == nil {
				oracle := m.(*pb.Oracle)
				if c, err = compileOracle(oracle); err != nil {
					err = fmt.Errorf("error while compiling oracle %d: %s", oracle.Id, err)
				} else {
					cache.Add(oracle.Id, c)
				}
			}
		})

		if err != nil {
			return nil, err
		}
	}

	return &Service{
		started: time.Now(),
		pid:     uint64(os.Getpid()),
		uid:     uint64(os.Getuid()),
		argv:    os.Args,
		records: records,
		oracles: oracles,
		cache:   cache,
	}, nil
}

// Info returns a *pb.ServerInfo object with various realtime information
// about the service and its runtime.
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

// Run executes a compiled oracle given its identifier and the arguments
// in the *pb.Call object.
func (s *Service) Run(ctx context.Context, call *pb.Call) (*pb.CallResponse, error) {
	compiled := s.cache.Get(call.OracleId)
	if compiled == nil {
		return errCallResponse("oracle %d not found.", call.OracleId), nil
	}

	// TODO: here the returned context could be used to queue and
	// finalize any write operations the oracle generated during
	// execution, making it transactional.
	_, raw, err := compiled.RunWithContext(s.records, call.Args)
	if err != nil {
		return errCallResponse("error while running oracle %d: %s", call.OracleId, err), nil
	}

	size := len(raw)
	compressed := false

	if size > gzipResponseSize {
		var buf bytes.Buffer
		if w, err := gzip.NewWriterLevel(&buf, gzip.BestCompression); err == nil {
			w.Write(raw)
			w.Close()

			compressed = true
			raw = buf.Bytes()
		}
	}

	return &pb.CallResponse{
		Success: true,
		Data: &pb.Data{
			Compressed: compressed,
			Payload:    raw,
		},
	}, nil
}
