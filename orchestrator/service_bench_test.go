package orchestrator

import (
	"context"
	"errors"
	"fmt"
	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/service"
	"google.golang.org/grpc"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

const vectorSize = 475
const numBenchRecords = 1024

func genBenchRecord() *pb.Record {
	rec := &pb.Record{}
	rec.Data = make([]float32, 0, vectorSize)

	for i := 0; i < vectorSize; i++ {
		f := rand.Float32()
		rec.Data = append(rec.Data, f)
	}
	return rec
}

func genBenchRecords() chan *pb.Record {
	ch := make(chan *pb.Record)

	go func() {
		for i := 0; i < numBenchRecords; i++ {
			ch <- genBenchRecord()
		}
		close(ch)
	}()

	return ch
}

type nodeSetup struct {
	server   *grpc.Server
	svc      *service.Service
	dataPath string
}

type orchestratorSetup struct {
	server            *grpc.Server
	svc               *MuxService
	updaterCancelFunc context.CancelFunc
}

type networkSetup struct {
	nodes         []nodeSetup
	orchestrators []orchestratorSetup
	oracleId      uint64
}

// setup a network of nodes and orchestrators for tests and benches
func setupNetwork(numNodes, numOrchestrators int) (setup networkSetup, err error) {
	defer func(e *error) {
		if *e == nil {
			return
		}
		cleanupNetwork(&setup)
	}(&err)

	if numNodes > 1 && numOrchestrators < 1 {
		panic("Orchestrator required for multiple nodes")
	}

	SetCommunicationTimeout(time.Second)

	var dir string
	nodesStr := &strings.Builder{}
	useTestFolder := false

	if stat, err := os.Stat(testFolder); err == nil && numNodes == 1 && stat.IsDir() {
		useTestFolder = true
		dir = testFolder
	}

	for i := 0; i < numNodes; i++ {
		if !useTestFolder {
			dir, err = ioutil.TempDir("", "")
			if err != nil {
				return
			}
			for _, childDir := range []string{"data", "oracles"} {
				err = os.Mkdir(filepath.Join(dir, childDir), 0755)
				if err != nil {
					return
				}
			}
		}
		n := &nodeSetup{}
		n.dataPath = dir
		port := uint32(12345 + i)
		n.server, n.svc, err = spawnNodeErr(port, dir)
		if err != nil {
			return
		}
		setup.nodes = append(setup.nodes, *n)
		if nodesStr.Len() > 0 {
			nodesStr.WriteString(",")
		}
		_, err = fmt.Fprintf(nodesStr, "127.0.0.1:%d", port)
		if err != nil {
			return
		}
	}

	for i := 0; i < numOrchestrators; i++ {
		o := &orchestratorSetup{}
		o.server, o.svc, err = spawnOrchestratorErr(uint32(12345+numNodes+i), nodesStr.String())
		if err != nil {
			return
		}
		setup.orchestrators = append(setup.orchestrators, *o)
	}

	return
}

func setupPopulatedNetwork(numNodes, numOrchestrators int) (setup networkSetup, err error) {
	setup, err = setupNetwork(numNodes, numOrchestrators)
	if err != nil {
		return
	}
	defer func(e *error) {
		if *e == nil {
			return
		}
		cleanupNetwork(&setup)
	}(&err)

	var createRecord func(context.Context, *pb.Record) (*pb.RecordResponse, error)
	var createOracle func(context.Context, *pb.Oracle) (*pb.OracleResponse, error)

	if numOrchestrators == 0 {
		sum := setup.nodes[0].svc
		createRecord = sum.CreateRecord
		createOracle = sum.CreateOracle
	} else {
		ms := setup.orchestrators[0].svc
		createRecord = ms.CreateRecord
		createOracle = ms.CreateOracle
	}

	for r := range genBenchRecords() {
		var resp *pb.RecordResponse
		resp, err = createRecord(context.Background(), r)
		if err != nil {
			return
		}
		if !resp.Success {
			err = errors.New(resp.Msg)
			return
		}
	}

	arg := &pb.Oracle{}
	arg.Name = "findSimilar"
	arg.Code = `
function findSimilar(id, threshold) {
    var v = records.Find(id);
    if( v.IsNull() == true ) {
        return ctx.Error("Vector " + id + " not found.");
    }

    var results = {};
    records.AllBut(v).forEach(function(record){
        var similarity = v.Cosine(record);
        if( similarity >= threshold ) {
           results[record.ID] = similarity
        }
    });

    return results;
}`

	var resp *pb.OracleResponse

	resp, err = createOracle(context.Background(), arg)
	if err != nil {
		return
	}
	if !resp.Success {
		err = errors.New(resp.Msg)
		return
	}

	setup.oracleId, err = strconv.ParseUint(resp.Msg, 10, 64)

	return
}

func cleanupNetwork(ns *networkSetup) {
	for _, o := range ns.orchestrators {
		if o.server != nil {
			o.server.Stop()
		}
	}
	for _, n := range ns.nodes {
		if n.server != nil {
			n.server.Stop()
		}
		os.RemoveAll(n.dataPath)
	}
}

// benchmark a network composed of only 1 sum node ( previous not distributed approach )
func BenchmarkSingleSum(b *testing.B) {
	setup, err := setupPopulatedNetwork(1, 0)
	if err != nil {
		panic(err)
	}
	defer cleanupNetwork(&setup)

	// Bench time!

	call := &pb.Call{}
	call.OracleId = setup.oracleId
	call.Args = []string{"id", "0.5"}
	sum := setup.nodes[0].svc

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		call.Args[0] = strconv.FormatUint(uint64(i%numBenchRecords)+1, 10)
		resp, err := sum.Run(context.Background(), call)
		if err != nil {
			panic(err)
		}
		if !resp.Success {
			panic(resp.Msg)
		}
	}
}

// 2 nodes and 1 orchestrator
func BenchmarkDoubleSum(b *testing.B) {
	setup, err := setupPopulatedNetwork(2, 1)
	if err != nil {
		panic(err)
	}
	defer cleanupNetwork(&setup)

	// Bench time!

	call := &pb.Call{}
	call.OracleId = setup.oracleId
	call.Args = []string{"id", "0.5"}
	ms := setup.orchestrators[0].svc

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		call.Args[0] = strconv.FormatUint(uint64(i%numBenchRecords)+1, 10)
		resp, err := ms.Run(context.Background(), call)
		if err != nil {
			panic(err)
		}
		if !resp.Success {
			panic(resp.Msg)
		}
	}
}

// 4 nodes and 1 orchestrator
func BenchmarkTetraSum(b *testing.B) {
	setup, err := setupPopulatedNetwork(4, 1)
	if err != nil {
		panic(err)
	}
	defer cleanupNetwork(&setup)

	// Bench time!

	call := &pb.Call{}
	call.OracleId = setup.oracleId
	call.Args = []string{"id", "0.5"}
	ms := setup.orchestrators[0].svc

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		call.Args[0] = strconv.FormatUint(uint64(i%numBenchRecords)+1, 10)
		resp, err := ms.Run(context.Background(), call)
		if err != nil {
			panic(err)
		}
		if !resp.Success {
			panic(resp.Msg)
		}
	}
}

// $(nproc) nodes and 1 orchestrator
func BenchmarkNprocSum(b *testing.B) {
	setup, err := setupPopulatedNetwork(runtime.NumCPU(), 1)
	if err != nil {
		panic(err)
	}
	defer cleanupNetwork(&setup)

	// Bench time!

	call := &pb.Call{}
	call.OracleId = setup.oracleId
	call.Args = []string{"id", "0.5"}
	ms := setup.orchestrators[0].svc

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		call.Args[0] = strconv.FormatUint(uint64(i%numBenchRecords)+1, 10)
		resp, err := ms.Run(context.Background(), call)
		if err != nil {
			panic(err)
		}
		if !resp.Success {
			panic(resp.Msg)
		}
	}
}
