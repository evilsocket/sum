package main

import (
	"context"
	"fmt"
	. "github.com/evilsocket/sum/proto"
	log "github.com/sirupsen/logrus"
)

func errNodeResponse(format string, args ...interface{}) *NodeResponse {
	return &NodeResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
}

func (ms *MuxService) AddNode(ctx context.Context, addr *ByAddr) (*NodeResponse, error) {
	n, err := createNode(addr.Address)
	if err != nil {
		return errNodeResponse("Cannot create node: %v", err), nil
	}

	ms.nodesLock.Lock()
	defer ms.nodesLock.Unlock()

	ms.nodes = append(ms.nodes, n)

	if err := ms.solveAllConflictsInTheWorld(); err != nil {
		log.Errorf("Cannot solve conflicts after adding node %d: %v", n.ID, err)
	}

	ms.balance()
	ms.stealOraclesFromNode(n)

	st := n.Status()
	mn := &Node{Id: uint64(n.ID), Name: n.Name, Info: &st}

	return &NodeResponse{Success: true, Nodes: []*Node{mn}}, nil
}

func (ms *MuxService) ListNodes(context.Context, *Empty) (*NodeResponse, error) {
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

func (ms *MuxService) DeleteNode(ctx context.Context, id *ById) (*NodeResponse, error) {

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

	ms.balance()

	return &NodeResponse{Success: true}, nil
}
