package master

import (
	"context"
	"fmt"
	. "github.com/evilsocket/sum/proto"
)

// add a node to control
func (ms *Service) AddNode(ctx context.Context, addr *ByAddr) (*NodeResponse, error) {
	n, err := CreateNode(addr.Address, addr.CertFile)
	if err != nil {
		return errNodeResponse("Cannot create node: %v", err), nil
	}

	ms.nodesLock.Lock()
	defer ms.nodesLock.Unlock()

	ms.setNextIdIfHigher(n.Status().NextRecordId)

	n.ID = ms.nextNodeId
	ms.nodes = append(ms.nodes, n)

	ms.nextNodeId++

	go ms.updateConfig()

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

	nRecords := n.Status().Records
	nNodes := uint64(len(ms.nodes))

	if nNodes > 0 && nRecords > 0 {
		perNode := nRecords / nNodes
		remainder := nRecords % nNodes

		for i, n1 := range ms.nodes {
			target := perNode

			if uint64(i) < remainder {
				target++
			}

			ms.transfer(n, n1, int64(target))
		}
	}

	return &NodeResponse{Success: true}, nil
}
