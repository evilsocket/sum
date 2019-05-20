package master

import (
	"context"
	"fmt"
	"github.com/evilsocket/islazy/log"
	. "github.com/evilsocket/sum/proto"
)

// steal oracles from underlying nodes
func (ms *Service) stealOracles() {
	for _, n := range ms.nodes {
		ms.stealOraclesFromNode(n)
	}
}

// steal an oracle form another node
func (ms *Service) stealOraclesFromNode(n *NodeInfo) {
	nOracles := n.Status().Oracles

	for i := uint64(0); i < nOracles; i++ {
		if err := ms.deployAgentSmith(n, i+1); err != nil {
			log.Error("Failed to absorb oracle: %v", err)
		}
	}

	go n.UpdateStatus()
}

func (ms *Service) doIHaveThisOracle(oracle *Oracle) bool {
	ms.cageLock.RLock()
	defer ms.cageLock.RUnlock()

	for _, raccoon := range ms.raccoons {
		if raccoon.IsEqualTo(oracle) {
			return true
		}
	}

	return false
}

// send agent Smith to absorb the oracle
func (ms *Service) deployAgentSmith(n *NodeInfo, oracleId uint64) error {
	ctx, cf := newCommContext()
	defer cf()

	resp, err := n.Client.ReadOracle(ctx, &ById{Id: oracleId})
	if err != nil || !resp.Success {
		return fmt.Errorf("unable to read oracle #%d from node %d: %v",
			oracleId, n.ID, getErrorMessage(err, resp))
	}

	oracle := resp.Oracle

	if !ms.doIHaveThisOracle(oracle) {
		if resp1, err := ms.CreateOracle(context.Background(), &Oracle{Code: oracle.Code, Name: oracle.Name}); err != nil || !resp1.Success {
			return fmt.Errorf("unable to load oracle #%d (%s) from node %d: %v",
				oracleId, oracle.Name, n.ID, getErrorMessage(err, resp1))
		}
	}

	if resp2, err := n.Client.DeleteOracle(ctx, &ById{Id: oracleId}); err != nil || !resp2.Success {
		log.Warning("cannot delete oracle #%d (%s) from node %d: %v",
			oracleId, oracle.Name, n.ID, getErrorMessage(err, resp2))
	}

	return nil
}
