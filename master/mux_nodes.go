package master

import (
	"context"
	"fmt"
	. "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/islazy/log"
)

// add a node to control
func (ms *Service) AddNode(ctx context.Context, addr *ByAddr) (*NodeResponse, error) {
	n, err := CreateNode(addr.Address, addr.CertFile)
	if err != nil {
		return errNodeResponse("Cannot create node: %v", err), nil
	}

	ms.nodesLock.Lock()
	defer ms.nodesLock.Unlock()
	ms.recordsLock.Lock()
	defer ms.recordsLock.Unlock()

	n.ID = ms.nextNodeId
	ms.nodes = append(ms.nodes, n)

	ms.nextNodeId++

	if err := ms.solveAllConflictsInTheWorld(); err != nil {
		log.Error("Cannot solve conflicts after adding node %d: %v", n.ID, err)
	} else {
		go ms.updateConfig()
	}

	ms.balance()
	ms.stealOraclesFromNode(n)

	return &NodeResponse{Success: true, Msg: fmt.Sprintf("%d", n.ID)}, nil
}

// list all controlled nodes
func (ms *Service) ListNodes(context.Context, *Empty) (*NodeResponse, error) {
	res := &NodeResponse{Success: true}

	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	for _, n := range ms.nodes {
		st := n.Status()
		mn := &Node{Id: uint64(n.ID), Name: n.Name, Info: &st}
		res.Nodes = append(res.Nodes, mn)
	}

	return res, nil
}

// delete a specified node
func (ms *Service) DeleteNode(ctx context.Context, id *ById) (*NodeResponse, error) {
	ms.nodesLock.Lock()
	defer ms.nodesLock.Unlock()

	var i int
	var n *NodeInfo

	for i, n = range ms.nodes {
		if uint64(n.ID) == id.Id {
			break
		}
	}

	if uint64(n.ID) != id.Id {
		return errNodeResponse("node %d not found.", id.Id), nil
	}

	l := len(ms.nodes)
	ms.nodes[i] = ms.nodes[l-1]
	ms.nodes[l-1] = nil
	ms.nodes = ms.nodes[:l-1]

	go ms.updateConfig()

	//TODO: balance on the fly
	if len(ms.nodes) > 0 {
		transfer(n, ms.nodes[0], int64(len(n.RecordIds)))
	}
	ms.balance()

	return &NodeResponse{Success: true}, nil
}
