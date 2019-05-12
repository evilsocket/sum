package main

import (
	"context"
	"fmt"
	. "github.com/evilsocket/sum/proto"
	log "github.com/sirupsen/logrus"
)

// steal oracles from underlying nodes
func (ms *MuxService) stealOracles() {
	for _, n := range ms.nodes {
		ms.stealOraclesFromNode(n)
	}
}

// steal an oracle form another node
func (ms *MuxService) stealOraclesFromNode(n *NodeInfo) {
	n.RLock()
	defer n.RUnlock()

	for i := uint64(0); i < n.status.Oracles; i++ {
		if err := ms.deployAgentSmith(n, i+1); err != nil {
			log.Errorf("Failed to absorb oracle: %v", err)
		}
	}
}

// send agent Smith to absorb the oracle
func (ms *MuxService) deployAgentSmith(n *NodeInfo, oracleId uint64) error {
	ctx, cf := newCommContext()
	defer cf()

	resp, err := n.Client.ReadOracle(ctx, &ById{Id: oracleId})
	if err != nil || !resp.Success {
		return fmt.Errorf("unable to read oracle #%d from node %d: %v",
			oracleId, n.ID, getErrorMessage(err, resp))
	}

	oracle := resp.Oracle
	//TODO: check for duplicates.

	if resp1, err := ms.CreateOracle(context.Background(), &Oracle{Code: oracle.Code, Name: oracle.Name}); err != nil || !resp1.Success {
		return fmt.Errorf("unable to load oracle #%d (%s) from node %d: %v",
			oracleId, oracle.Name, n.ID, getErrorMessage(err, resp1))
	} else if resp2, err := n.Client.DeleteOracle(ctx, &ById{Id: oracleId}); err != nil || !resp2.Success {
		log.Warnf("cannot delete oracle #%d (%s) from node %d: %v",
			oracleId, oracle.Name, n.ID, getErrorMessage(err, resp2))
	}

	return nil
}
