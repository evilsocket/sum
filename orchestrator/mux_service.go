package main

import (
	"github.com/evilsocket/sum/service"
	"github.com/robertkrimen/otto"
	log "github.com/sirupsen/logrus"
	"os"
	"sync"
	"time"
)

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
	// vm pool
	vmPool *service.ExecutionPool

	// stats

	// start time
	started time.Time
	// pid
	pid uint64
	// uid
	uid uint64
}

func NewMuxService(nodes []*NodeInfo) (*MuxService, error) {
	ms := &MuxService{
		nextId:        1,
		nextRaccoonId: 1,
		recId2node:    make(map[uint64]*NodeInfo),
		nodes:         nodes[:],
		raccoons:      make(map[uint64]*astRaccoon),
		vmPool:        service.CreateExecutionPool(otto.New()),
		started:       time.Now(),
		pid:           uint64(os.Getpid()),
		uid:           uint64(os.Getuid()),
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

func (ms *MuxService) UpdateNodes() {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	for _, n := range ms.nodes {
		n.UpdateStatus()
	}
}

func (ms *MuxService) AddNode(n *NodeInfo) {
	ms.nodesLock.Lock()
	defer ms.nodesLock.Unlock()

	ms.nodes = append(ms.nodes, n)

	if err := ms.solveAllConflictsInTheWorld(); err != nil {
		log.Errorf("Cannot solve conflicts after adding node %d: %v", n.ID, err)
	}

	ms.balance()
	ms.stealOraclesFromNode(n)
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
