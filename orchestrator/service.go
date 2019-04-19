package main

import (
	"context"
	"fmt"
	. "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/wrapper"
	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/file"
	"github.com/robertkrimen/otto/parser"
	log "github.com/sirupsen/logrus"
	"sort"
	"strings"
	"sync"
)

func errRecordResponse(format string, args ...interface{}) *RecordResponse {
	return &RecordResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
}

func errOracleResponse(format string, args ...interface{}) *OracleResponse {
	return &OracleResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
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

type astRaccoon struct {
	src          string
	firstArgName string
	callNodes    []ast.Node
}

func (a *astRaccoon) PatchCode(record *Record) (newCode string, err error) {
	var compressed string
	shift := file.Idx(0)
	newCode = a.src

	if compressed, err = wrapper.RecordToCompressedText(record); err != nil {
		return
	}

	newRecord := fmt.Sprintf("records.New('%s')", compressed)
	newRecordLen := file.Idx(len(newRecord))

	for _, n := range a.callNodes {
		idx0 := n.Idx0() + shift - 1
		idx1 := n.Idx1() + shift - 1
		newCode = newCode[:idx0] + newRecord + newCode[idx1:]
		shift += newRecordLen - (idx1 - idx0)
	}

	return
}

func (a *astRaccoon) Enter(n ast.Node) ast.Visitor {
	if _, ok := n.(*ast.CallExpression); ok {
		idx0 := n.Idx0() - 1
		idx1 := n.Idx1() - 1
		callStr := a.src[idx0:idx1]
		callStr = strings.Join(strings.Fields(callStr), "")
		if callStr == ("records.Find(" + a.firstArgName + ")") {
			a.callNodes = append(a.callNodes, n)
			return nil
		}
	}

	return a
}

func (a *astRaccoon) Exit(n ast.Node) {}

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

	// 2. make a list of nodes that invoke records.Find(id)

	//TODO check parameter list
	raccoon := &astRaccoon{src: arg.Code[:], firstArgName: oracleFunction.Function.ParameterList.List[0].Name}
	ast.Walk(raccoon, oracleFunction.Function)

	//TODO: store raccoon

	// -- Runtime
	// 3. replace records.Find(id) with current vector, build js, send it to nodes, run it
	// 4. parse json output and use the merge function if provided ( required for scalars )
	panic("not implemented yet")

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
	panic("not implemented yet")
}
func (ms *MuxService) Info(ctx context.Context, arg *Empty) (*ServerInfo, error) {
	panic("not implemented yet")
}
