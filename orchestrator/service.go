package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	. "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/service"
	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/parser"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func errRecordResponse(format string, args ...interface{}) *RecordResponse {
	return &RecordResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
}

func errOracleResponse(format string, args ...interface{}) *OracleResponse {
	return &OracleResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
}

func errCallResponse(format string, args ...interface{}) *CallResponse {
	return &CallResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
}

type MuxService struct {
	// control access to `nodes`
	nodesLock sync.RWMutex
	// currently available nodes
	nodes []*NodeInfo
	// control access to `nextId`
	idLock sync.RWMutex
	// id of the next record
	nextId uint64
	// map a record to its containing node
	recId2node map[uint64]*NodeInfo
	// control access to `raccoons`
	cageLock sync.RWMutex
	// raccoons ready to mess with messy JS code
	raccoons map[uint64]*astRaccoon
	// id of the next raccoon
	nextRaccoonId uint64
}

func NewMuxService(nodes []*NodeInfo) *MuxService {
	ms := &MuxService{nextId: 1, recId2node: make(map[uint64]*NodeInfo), nodes: make([]*NodeInfo, 0)}
	ms.nodes = append(ms.nodes, nodes...) // solves `nil` slice argument
	ms.balance()
	return ms
}

func (ms *MuxService) UpdateNodes() {
	ms.nodesLock.Lock()
	defer ms.nodesLock.Unlock()

	for _, n := range ms.nodes {
		n.UpdateStatus()
	}
}

func (ms *MuxService) AddNode(n *NodeInfo) {
	ms.nodesLock.Lock()
	defer ms.nodesLock.Unlock()

	ms.nodes = append(ms.nodes, n)

	ms.balance()
}

func (ms *MuxService) findNextAvailableId() uint64 {
	ms.idLock.Lock()
	defer ms.idLock.Unlock()

	for {
		found := false
		for _, n := range ms.nodes {
			if n.RecordIds[ms.nextId] {
				found = true
				break
			}
		}
		if !found {
			return ms.nextId
		}
		ms.nextId++
	}
}

func (ms *MuxService) CreateRecord(ctx context.Context, record *Record) (*RecordResponse, error) {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	var lowestRecords *uint64
	var targetNode *NodeInfo

	for _, n := range ms.nodes {
		st := n.Status()
		if lowestRecords == nil || st.Records < *lowestRecords {
			lowestRecords = &st.Records
			targetNode = n
		}
	}

	if targetNode == nil {
		return errRecordResponse("No nodes available, try later"), nil
	}

	record.Id = ms.findNextAvailableId()
	resp, err := targetNode.Client.CreateRecord(ctx, record)

	if err == nil && resp.Success {
		ms.nextId++
		targetNode.RecordIds[record.Id] = true
		ms.recId2node[record.Id] = targetNode
	}

	return resp, err
}

func (ms *MuxService) UpdateRecord(ctx context.Context, arg *Record) (*RecordResponse, error) {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	if n, found := ms.recId2node[arg.Id]; !found {
		return errRecordResponse("Record %d not found", arg.Id), nil
	} else {
		return n.Client.UpdateRecord(ctx, arg)
	}
}

func (ms *MuxService) ReadRecord(ctx context.Context, arg *ById) (*RecordResponse, error) {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	if n, found := ms.recId2node[arg.Id]; !found {
		return errRecordResponse("Record %d not found", arg.Id), nil
	} else {
		return n.Client.ReadRecord(ctx, arg)
	}
}

func (ms *MuxService) ListRecords(ctx context.Context, arg *ListRequest) (*RecordListResponse, error) {
	id2node := make(map[uint]*NodeInfo)
	workerInputs := make(map[uint]chan uint64)
	workerOutputs := make(chan *Record)

	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	for _, n := range ms.nodes {
		n.RLock()
		defer n.RUnlock() // defer: ensure consistent data across this whole function
		workerInputs[n.ID] = make(chan uint64)
	}

	total := uint64(len(ms.recId2node))
	pages := total / arg.PerPage
	if total%pages > 0 {
		pages++
	}
	start := arg.PerPage * (arg.Page - 1)
	end := start + arg.PerPage
	if end > total {
		end = total
	}
	resp := &RecordListResponse{Total: total, Pages: pages, Records: make([]*Record, 0, end-start)}

	if total == 0 || end == start {
		return resp, nil
	}

	//NB: can be improved by spawning more workers per node, each one with a different connection
	worker := func(n *NodeInfo, ch <-chan uint64, outch chan<- *Record) {
		for id := range ch {
			ctx, _ := newCommContext()
			resp, err := n.Client.ReadRecord(ctx, &ById{Id: id})
			if err != nil || !resp.Success {
				log.Errorf("Unable to read record %d on node %d: %v", id, n.ID, getTheFuckingErrorMessage(err, resp))
			} else {
				outch <- resp.Record
			}
		}
	}

	for nId, input := range workerInputs {
		go worker(id2node[nId], input, workerOutputs)
	}

	for id, n := range ms.recId2node {
		workerInputs[n.ID] <- id
	}

	for r := range workerOutputs {
		resp.Records = append(resp.Records, r)
	}

	sort.Slice(resp.Records, func(i, j int) bool {
		return resp.Records[i].Id < resp.Records[j].Id
	})

	return resp, nil
}

func (ms *MuxService) DeleteRecord(ctx context.Context, arg *ById) (*RecordResponse, error) {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	if n, found := ms.recId2node[arg.Id]; !found {
		return errRecordResponse("Record %d not found", arg.Id), nil
	} else {
		return n.Client.DeleteRecord(ctx, arg)
	}
}

func (ms *MuxService) FindRecords(ctx context.Context, arg *ByMeta) (*FindResponse, error) {
	panic("not implemented yet")
}

func (ms *MuxService) CreateOracle(ctx context.Context, arg *Oracle) (*OracleResponse, error) {
	functionList := make([]*ast.FunctionDeclaration, 0)
	// 1. parse the AST ( FunctionDeclaration.FunctionLiteral.Body, walk over it

	program, err := parser.ParseFile(nil, "", arg.Code, 0)
	if err != nil {
		return errOracleResponse("Cannot parse oracle: %v", err), nil
	}

	for _, d := range program.DeclarationList {
		if fd, ok := d.(*ast.FunctionDeclaration); ok {
			functionList = append(functionList, fd)
		}
	}

	if len(functionList) == 0 {
		return errOracleResponse("No function provided"), nil
	}

	oracleFunction := functionList[0]

	// 2. make a list of nodes that invoke records.Find(anyArg)

	raccoon := NewAstRaccoon(arg.Code, oracleFunction.Function)
	raccoon.Name = arg.Name

	// store the raccoon

	ms.cageLock.Lock()
	defer ms.cageLock.Unlock()

	raccoon.ID = ms.nextRaccoonId
	ms.nextRaccoonId++

	ms.raccoons[raccoon.ID] = raccoon

	return &OracleResponse{Success: true, Msg: fmt.Sprintf("%d", raccoon.ID)}, nil
}

func (ms *MuxService) UpdateOracle(ctx context.Context, arg *Oracle) (*OracleResponse, error) {
	panic("not implemented yet")
}
func (ms *MuxService) ReadOracle(ctx context.Context, arg *ById) (*OracleResponse, error) {
	panic("not implemented yet")
}
func (ms *MuxService) FindOracle(ctx context.Context, arg *ByName) (*OracleResponse, error) {
	panic("not implemented yet")
}
func (ms *MuxService) DeleteOracle(ctx context.Context, arg *ById) (*OracleResponse, error) {
	panic("not implemented yet")
}
func (ms *MuxService) Run(ctx context.Context, arg *Call) (*CallResponse, error) {

	// NB: always keep this order of locking
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()
	ms.cageLock.RLock()
	defer ms.cageLock.RUnlock()

	raccoon, found := ms.raccoons[arg.OracleId]
	if !found {
		return errCallResponse("Oracle %d not found", arg.OracleId), nil
	}

	// 1. Find the record the oracle is working on

	resolvedRecords := make([]*Record, len(arg.Args)) // fill with nil

	for i, a := range arg.Args {
		if !raccoon.IsParameterPositionARecordLookup(i) {
			continue
		}

		recId, err := strconv.ParseUint(a, 10, 64)
		if err != nil {
			return errCallResponse("Unable to parse record id form parameter #%d: %v", i, err), nil
		}
		node, found := ms.recId2node[recId]
		if !found {
			return errCallResponse("Record %d not found", recId), nil
		}
		record, err := node.Client.ReadRecord(ctx, &ById{Id: recId})
		if err != nil || !record.Success {
			return errCallResponse("Unable to retrieve record %d form node %d: %v",
				recId, node.ID, getTheFuckingErrorMessage(err, record)), nil
		}
		resolvedRecords[i] = record.Record
	}

	// 2. substitute all the calls to records.Find(...) with their resolved record

	newCode, err := raccoon.PatchCode(resolvedRecords)
	if err != nil {
		return errCallResponse("Unable to patch JS code: %v", err), nil
	}

	// 3. create the modified oracle on all nodes
	node2oracleId := make(map[*NodeInfo]uint64)
	newOracle := &Oracle{Code: newCode, Name: raccoon.Name}

	// cleanup created oracles
	defer func() {
		for n, oId := range node2oracleId {
			resp, err := n.Client.DeleteOracle(ctx, &ById{Id: oId})
			if err != nil || !resp.Success {
				log.Warnf("Unable to delete temporary oracle %d on node %d: %v",
					oId, n.ID, getTheFuckingErrorMessage(err, resp))
			}
		}
	}()

	worker := func(wg sync.WaitGroup, n *NodeInfo, okChan chan<- interface{}, errChan chan<- string) {
		defer wg.Done()
		resp, err := n.Client.CreateOracle(ctx, newOracle)
		if err != nil || !resp.Success {
			errChan <- getTheFuckingErrorMessage(err, resp)
			return
		}
		oId, err := strconv.ParseUint(resp.Msg, 10, 64)
		if err != nil {
			errChan <- fmt.Sprintf("unable to parse oracleId string '%s': %v", resp.Msg, err)
			return
		}
		node2oracleId[n] = oId
		resp1, err := n.Client.Run(ctx, &Call{OracleId: oId, Args: arg.Args})
		if err != nil || !resp1.Success {
			errChan <- getTheFuckingErrorMessage(err, resp1)
			return
		}
		if resp1.Data.Compressed {
			if r, err := gzip.NewReader(bytes.NewReader(resp1.Data.Payload)); err != nil {
				errChan <- err.Error()
				return
			} else if resp1.Data.Payload, err = ioutil.ReadAll(r); err != nil {
				errChan <- err.Error()
				return
			}
		}
		var res interface{}
		if err = json.Unmarshal(resp1.Data.Payload, &res); err != nil {
			errChan <- err.Error()
			return
		}
		okChan <- res
	}

	errorChan := make(chan string)
	resultChan := make(chan interface{})
	wg := sync.WaitGroup{}
	wg.Add(len(ms.nodes))

	for _, n := range ms.nodes {
		go worker(wg, n, resultChan, errorChan)
	}

	wg.Wait()

	close(errorChan)
	close(resultChan)

	errs := make([]string, 0)
	var resultType *reflect.Type
	var mergedResults interface{} = nil

	for err := range errorChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errCallResponse("Error from nodes: %s", strings.Join(errs, ", ")), nil
	}

	for res := range resultChan {
		t := reflect.TypeOf(res)
		if resultType == nil {
			resultType = &t
		} else if *resultType != t {
			return errCallResponse("Heterogeneous results: prior results had type %v, this one has type %v", *resultType, t), nil
		}

		//TODO: custom merge functions
		//FIXME: this shall be moved to a separate function
		if t.Kind() == reflect.Map {
			if mergedResults == nil {
				mergedResults = make(map[interface{}]interface{})
			}
			mr := mergedResults.(map[interface{}]interface{})
			for k, v := range res.(map[interface{}]interface{}) {
				if v1, exist := mr[k]; exist {
					return errCallResponse("Merge conflict: multiple results define key %v: oldValue='%v', newValue='%v'", k, v1, v), nil
				}
				mr[k] = v
			}
		} else if t.Kind() == reflect.Array {
			if mergedResults == nil {
				mergedResults = make([]interface{}, 0)
			}
			mr := mergedResults.([]interface{})
			for _, v := range res.([]interface{}) {
				mr = append(mr, v)
			}
		} else {
			return errCallResponse("Type %v is not supported for auto-merge, please provide a custom merge function", t), nil
		}
	}

	if raw, err := json.Marshal(mergedResults); err != nil {
		return errCallResponse("Unable to marshal result: %v", err), nil
	} else {
		return &CallResponse{Success: true, Msg: "", Data: service.BuildPayload(raw)}, nil
	}
}

func (ms *MuxService) Info(ctx context.Context, arg *Empty) (*ServerInfo, error) {
	panic("not implemented yet")
}
