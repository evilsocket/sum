package orchestrator

import (
	"fmt"
	"github.com/evilsocket/sum/service"
	"github.com/robertkrimen/otto"
	"os"
	"sync"
	"time"
)

// A Service that multiplexes sum's workload
// among multiple sum instances
type MuxService struct {
	// control access to `nodes` and `nextNodeId`
	nodesLock sync.RWMutex
	// currently available nodes
	nodes []*NodeInfo
	// next node id
	nextNodeId uint
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
	// vm pool
	vmPool *service.ExecutionPool

	// stats

	// start time
	started time.Time
	// pid
	pid uint64
	// uid
	uid uint64
	// credentials path
	credsPath string
	// listening address
	address string
	// configuration file path
	configFile string
}

// create a new MuxService that manage the given nodes
func NewMuxService(nodes []*NodeInfo, credsPath, address string) (*MuxService, error) {
	ms := &MuxService{
		nextId:        1,
		nextRaccoonId: 1,
		nextNodeId:    uint(len(nodes) + 1),
		recId2node:    make(map[uint64]*NodeInfo),
		nodes:         nodes[:],
		raccoons:      make(map[uint64]*astRaccoon),
		vmPool:        service.CreateExecutionPool(otto.New()),
		started:       time.Now(),
		pid:           uint64(os.Getpid()),
		uid:           uint64(os.Getuid()),
		credsPath:     credsPath,
		address:       address,
	}

	if err := ms.solveAllConflictsInTheWorld(); err != nil {
		return nil, err
	}

	for _, n := range nodes {
		for rId := range n.RecordIds {
			ms.recId2node[rId] = n
		}
	}

	ms.balance()
	ms.stealOracles()

	return ms, nil
}

// create a new MuxService with the configuration from at `configPath`
func NewMuxServiceFromConfig(configPath, credsPath, address string) (*MuxService, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("cannot load config from '%s': %v", configPath, err)
	}

	nodes := make([]*NodeInfo, 0, len(cfg.Nodes))
	for _, nc := range cfg.Nodes {
		n, err := CreateNode(nc.Address, nc.CertFile)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}

	ms, err := NewMuxService(nodes, credsPath, address)
	if err == nil {
		ms.configFile = configPath
	}
	return ms, err
}

// update the managed nodes's status
func (ms *MuxService) UpdateNodes() {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	for _, n := range ms.nodes {
		n.UpdateStatus()
	}
}

// find the next available node ID
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

func (ms *MuxService) NumRecords() int {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	return len(ms.recId2node)
}

func (ms *MuxService) NumOracles() int {
	ms.cageLock.RLock()
	defer ms.cageLock.RUnlock()

	return len(ms.raccoons)
}
