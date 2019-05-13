package orchestrator

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/service"
	log "github.com/sirupsen/logrus"
	. "github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func TestCreateOracle(t *testing.T) {
	ms, err := NewMuxService([]*NodeInfo{}, "", "")
	Nil(t, err)

	arg := &pb.Oracle{}
	arg.Name = "alakazam"
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

	resp, err := ms.CreateOracle(context.Background(), arg)
	Nil(t, err)
	True(t, resp.Success)
}

func TestAstRaccoon(t *testing.T) {
	code := `
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

	var x = records.Find(id);

    return results;
}`
	raccoon, err := NewAstRaccoon(code)
	Nil(t, err)

	r := &pb.Record{Id: 1, Meta: map[string]string{"key": "value"}, Data: []float32{0.1, 0.2, 0.3}}
	newCode, err := raccoon.PatchCode([]*pb.Record{r, nil})
	Nil(t, err)

	expected := strings.Replace(code, "records.Find(id)", "records.New('eJziYBTiOXvmjO3ZMz52s2bOtJPi4WLOTq0UYi1LzClNBQQAAP//qfgKpw==')", -1)
	Equal(t, expected, newCode)
}

func spawnNodeErr(port uint32, dataPath string) (*grpc.Server, *service.Service, error) {
	addr := fmt.Sprintf("localhost:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, err
	}
	svc, err := service.New(dataPath, "", "")
	if err != nil {
		listener.Close()
		return nil, nil, err
	}
	server := grpc.NewServer()
	pb.RegisterSumServiceServer(server, svc)
	pb.RegisterSumInternalServiceServer(server, svc)
	reflection.Register(server)

	go func() {
		if err := server.Serve(listener); err != nil {
			log.Errorf("Failed to serve gRPC server: %v", err)
		}
	}()

	return server, svc, nil
}

func spawnNode(t *testing.T, port uint32, dataPath string) (*grpc.Server, *service.Service) {
	server, svc, err := spawnNodeErr(port, dataPath)
	Nil(t, err)
	return server, svc
}

func spawnOrchestratorErr(port uint32, nodesStr string) (*grpc.Server, *MuxService, error) {
	nodes := make([]*NodeInfo, 0)

	if nodesStr != "" {
		for _, n := range strings.Split(nodesStr, ",") {
			node, err := CreateNode(n, "")
			if err != nil {
				return nil, nil, err
			}
			node.ID = uint(len(nodes) + 1)
			nodes = append(nodes, node)
		}
	}

	addr := fmt.Sprintf("localhost:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, err
	}

	ms, err := NewMuxService(nodes, "", addr)
	if err != nil {
		return nil, nil, err
	}

	ctx, cf := context.WithCancel(context.Background())
	go NodeUpdater(ctx, ms, time.Second)

	server := grpc.NewServer()
	pb.RegisterSumServiceServer(server, ms)
	reflection.Register(server)

	go func() {
		server.Serve(listener)
		cf()
	}()

	return server, ms, nil
}

func setupEmptyTmpFolder() (string, error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	for _, childDir := range []string{"data", "oracles"} {
		err = os.Mkdir(filepath.Join(dir, childDir), 0755)
		if err != nil {
			os.RemoveAll(dir)
			return "", err
		}
	}

	return dir, nil
}

func spawnOrchestrator(t *testing.T, port uint32, nodesStr string) (*grpc.Server, *MuxService) {
	server, ms, err := spawnOrchestratorErr(port, nodesStr)
	Nil(t, err)
	return server, ms
}

func TestDistributedRun(t *testing.T) {
	SetCommunicationTimeout(time.Second)

	dir1, err := setupEmptyTmpFolder()
	Nil(t, err)
	defer os.RemoveAll(dir1)
	dir2, err := setupEmptyTmpFolder()
	Nil(t, err)
	defer os.RemoveAll(dir2)

	node1, sum1 := spawnNode(t, 12345, dir1)
	defer node1.Stop()
	node2, sum2 := spawnNode(t, 12346, dir2)
	defer node2.Stop()

	master, ms := spawnOrchestrator(t, 12347, "localhost:12345,localhost:12346")
	defer master.Stop()

	// Test time!

	// create records

	rec1 := &pb.Record{Data: []float32{0.1, 0.2, 0.3}, Meta: map[string]string{"name": "1"}}
	rec2 := &pb.Record{Data: []float32{0.2, 0.4, 0.6}, Meta: map[string]string{"name": "2"}}

	resp, err := ms.CreateRecord(context.Background(), rec1)
	Nil(t, err)
	True(t, resp.Success)
	rec1Id, err := strconv.ParseUint(resp.Msg, 10, 64)
	Nil(t, err)

	resp, err = ms.CreateRecord(context.Background(), rec2)
	Nil(t, err)
	True(t, resp.Success)
	rec2Id, err := strconv.ParseUint(resp.Msg, 10, 64)
	Nil(t, err)

	NotEqual(t, rec1Id, rec2Id)

	// check distribution

	Equal(t, 1, sum1.NumRecords())
	Equal(t, 1, sum2.NumRecords())

	list1, err := sum1.ListRecords(context.Background(), &pb.ListRequest{PerPage: 1024, Page: 1})
	Nil(t, err)
	list2, err := sum2.ListRecords(context.Background(), &pb.ListRequest{PerPage: 1024, Page: 1})

	createdRecords := append(list1.Records, list2.Records...)
	Equal(t, 2, len(createdRecords))
	NotEqual(t, createdRecords[0].Id, createdRecords[1].Id)

	// create oracle

	code := `
function findDoubles(id) {
    var v = records.Find(id);
    if( v.IsNull() == true ) {
        return ctx.Error("Vector " + id + " not found.");
    }

    var results = [];
    records.AllBut(v).forEach(function(record){
		for (var i=0; i < 3; i++) {  
			if (record.Get(i) !== 2*v.Get(i)) { return; }
		}
		results.push(record.ID);
    });

    return results;
}`

	resp1, err := ms.CreateOracle(context.Background(), &pb.Oracle{Code: code, Name: "findDoubles"})
	Nil(t, err)
	True(t, resp1.Success)
	oId, err := strconv.ParseUint(resp1.Msg, 10, 64)
	Nil(t, err)

	// run oracle

	arg1 := fmt.Sprintf("%d", rec1Id)
	resp2, err := ms.Run(context.Background(), &pb.Call{Args: []string{arg1}, OracleId: oId})
	Nil(t, err)
	True(t, resp2.Success)

	if resp2.Data.Compressed {
		r, err := gzip.NewReader(bytes.NewReader(resp2.Data.Payload))
		Nil(t, err)
		resp2.Data.Payload, err = ioutil.ReadAll(r)
		Nil(t, err)
	}

	var res interface{}

	err = json.Unmarshal(resp2.Data.Payload, &res)
	Nil(t, err)

	// check result

	ary, ok := res.([]interface{})
	True(t, ok)
	Equal(t, 1, len(ary))
	resId, ok := ary[0].(float64)
	True(t, ok)
	Equal(t, rec2Id, uint64(resId))
}

func TestMergerFunction(t *testing.T) {
	SetCommunicationTimeout(time.Second)

	dir1, err := ioutil.TempDir("", "")
	Nil(t, err)
	defer os.RemoveAll(dir1)
	dir2, err := ioutil.TempDir("", "")
	Nil(t, err)
	defer os.RemoveAll(dir2)

	for _, baseDir := range []string{dir1, dir2} {
		for _, childDir := range []string{"data", "oracles"} {
			err = os.Mkdir(filepath.Join(baseDir, childDir), 0755)
			Nil(t, err)
		}
	}

	node1, _ := spawnNode(t, 12345, dir1)
	defer node1.Stop()
	node2, _ := spawnNode(t, 12346, dir2)
	defer node2.Stop()

	master, ms := spawnOrchestrator(t, 12347, "localhost:12345,localhost:12346")
	defer master.Stop()

	// Test time!

	// create records

	rec1 := &pb.Record{Data: []float32{0.1, 0.2, 0.3}, Meta: map[string]string{"name": "1"}}
	rec2 := &pb.Record{Data: []float32{0.2, 0.4, 0.6}, Meta: map[string]string{"name": "2"}}

	resp, err := ms.CreateRecord(context.Background(), rec1)
	Nil(t, err)
	True(t, resp.Success)
	rec1Id, err := strconv.ParseUint(resp.Msg, 10, 64)
	Nil(t, err)

	resp, err = ms.CreateRecord(context.Background(), rec2)
	Nil(t, err)
	True(t, resp.Success)
	rec2Id, err := strconv.ParseUint(resp.Msg, 10, 64)
	Nil(t, err)

	NotEqual(t, rec1Id, rec2Id)

	// create oracle

	code := `
function sumAllVectors() {
    var result = 0.0;
    records.All().forEach(function(record){
		for (var i=0; i < 3; i++) {
			result += record.Get(i);
		}
    });

    return result;
}
`

	// shall fail without a merger function

	resp1, err := ms.CreateOracle(context.Background(), &pb.Oracle{Code: code, Name: "sumAllVectors"})
	Nil(t, err)
	True(t, resp1.Success)
	oId, err := strconv.ParseUint(resp1.Msg, 10, 64)
	Nil(t, err)

	resp2, err := ms.Run(context.Background(), &pb.Call{Args: []string{}, OracleId: oId})
	Nil(t, err)
	False(t, resp2.Success)

	// add merger function

	code += `
function add(accumulator, a) { return accumulator + a; }

function mergeNodesResults(results) {
	return results.reduce(add);
}
`

	resp1, err = ms.CreateOracle(context.Background(), &pb.Oracle{Code: code, Name: "sumAllVectors"})
	Nil(t, err)
	True(t, resp1.Success)
	oId, err = strconv.ParseUint(resp1.Msg, 10, 64)
	Nil(t, err)

	// run oracle

	resp2, err = ms.Run(context.Background(), &pb.Call{Args: []string{}, OracleId: oId})
	Nil(t, err)
	True(t, resp2.Success)

	if resp2.Data.Compressed {
		r, err := gzip.NewReader(bytes.NewReader(resp2.Data.Payload))
		Nil(t, err)
		resp2.Data.Payload, err = ioutil.ReadAll(r)
		Nil(t, err)
	}

	var res interface{}

	err = json.Unmarshal(resp2.Data.Payload, &res)
	Nil(t, err)

	// check result

	val, ok := res.(float64)
	True(t, ok)
	InEpsilon(t, 1.8, val, 1e-6)
}

func TestAddNode(t *testing.T) {
	ns, err := setupPopulatedNetwork(2, 1)
	NoError(t, err)
	defer cleanupNetwork(&ns)

	dir, err := setupEmptyTmpFolder()
	NoError(t, err)

	node, sum := spawnNode(t, 12348, dir)
	defer node.Stop()

	Zero(t, sum.NumRecords())

	resp, err := ns.orchestrators[0].svc.AddNode(context.TODO(), &pb.ByAddr{Address: "127.0.0.1:12348"})
	NoError(t, err)
	True(t, resp.Success)

	id, err := strconv.ParseUint(resp.Msg, 10, 64)
	NoError(t, err)
	Equal(t, uint64(3), id)

	// check balancing

	for _, sumSetup := range ns.nodes {
		InDelta(t, sum.NumRecords(), sumSetup.svc.NumRecords(), 1)
	}
}

func TestListNodes(t *testing.T) {
	ns, err := setupPopulatedNetwork(3, 1)
	NoError(t, err)
	defer cleanupNetwork(&ns)

	resp, err := ns.orchestrators[0].svc.ListNodes(context.TODO(), &pb.Empty{})
	NoError(t, err)
	True(t, resp.Success)

	Equal(t, 3, len(resp.Nodes))
}

func TestDeleteNode(t *testing.T) {
	ns, err := setupPopulatedNetwork(2, 1)
	NoError(t, err)
	defer cleanupNetwork(&ns)

	ms := ns.orchestrators[0].svc

	nRecords := len(ms.recId2node)

	resp, err := ms.DeleteNode(context.TODO(), &pb.ById{Id: 1})
	NoError(t, err)
	True(t, resp.Success)

	Equal(t, 1, len(ms.nodes))
	Equal(t, nRecords, len(ms.recId2node))
}
