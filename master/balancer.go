package master

import (
	pb "github.com/evilsocket/sum/proto"

	"github.com/evilsocket/islazy/log"
)

// transfer nRecords from a node to another
func (ms *Service) transfer(fromNode, toNode *NodeInfo, nRecords int64) {
	log.Info("transferring %d records: %s -> %s ...", nRecords, fromNode.Name, toNode.Name)

	fromNode.Lock()
	toNode.Lock()
	defer fromNode.Unlock()
	defer toNode.Unlock()

	ctx, cf := newCommContext()
	defer cf()

	list, err := fromNode.Client.ListRecords(ctx, &pb.ListRequest{PerPage: uint64(nRecords), Page: 1})
	if err != nil {
		log.Error("Cannot get records from node %d: %v", fromNode.ID, err)
		return
	}

	log.Debug("Master[%s]: got %d records from node %s", ms.address, len(list.Records), fromNode.Name)

	resp, err := toNode.InternalClient.CreateRecordsWithId(ctx, &pb.Records{Records: list.Records})

	if err != nil || !resp.Success {
		log.Error("Unable to store records on node %d: %v", toNode.ID, getErrorMessage(err, resp))
		return
	}

	log.Debug("Master[%s]: created %d records on node %s", ms.address, len(list.Records), toNode.Name)

	delReq := &pb.RecordIds{Ids: make([]uint64, 0, nRecords)}
	toNode.status.Records += uint64(nRecords)

	for _, r := range list.Records {
		delReq.Ids = append(delReq.Ids, r.Id)

		if toNode.status.NextRecordId <= r.Id {
			toNode.status.NextRecordId = r.Id + 1
		}
	}

	resp1, err := fromNode.InternalClient.DeleteRecords(ctx, delReq)

	if err != nil || !resp1.Success {
		log.Error("Unable to delete records from node %d: %v", fromNode.ID, getErrorMessage(err, resp1))
		// keep going anyway
	} else {
		log.Debug("Master[%s]: deleted %d records from node %s", ms.address, len(delReq.Ids), fromNode.Name)
	}

	fromNode.status.Records -= uint64(nRecords)
}

// balance the load among nodes
func (ms *Service) balance() {
	log.Debug("Master[%s]: balancing...", ms.address)
	var totRecords = uint64(0)
	var nNodes = len(ms.nodes)

	for _, n := range ms.nodes {
		totRecords += n.Status().Records
	}

	if totRecords == 0 || nNodes == 0 {
		return
	}

	var targetRecordsPerNode = totRecords / uint64(nNodes)
	var reminder = int(totRecords % uint64(nNodes))
	// target amount of records
	var targets = make([]uint64, nNodes)
	// records that shall enter in the node
	var deltas = make([]int64, nNodes)
	var needsBalancing = false

	for i, n := range ms.nodes {
		targets[i] = targetRecordsPerNode
		if i < reminder {
			targets[i]++
		}
		deltas[i] = int64(targets[i]) - int64(n.Status().Records)

		// 5% hysteresis
		if !needsBalancing && deltas[i] > int64(targetRecordsPerNode/20) {
			needsBalancing = true
		}
	}

	if !needsBalancing {
		return
	}

	log.Info("balancing %d deltas ...", len(deltas))

	for i, delta := range deltas {
		// foreach node that need records
		if delta <= 0 {
			continue
		}
		// consume records from others
		for j, delta2 := range deltas {
			// foreach node that shall give records
			if delta2 >= 0 {
				continue
			}
			nRecords := -delta2
			if nRecords > delta {
				nRecords = delta
			}
			if nRecords == 0 {
				continue
			}
			ms.transfer(ms.nodes[j], ms.nodes[i], nRecords)
			delta -= nRecords
			deltas[i] -= nRecords
			deltas[j] += nRecords
			if delta == 0 {
				break
			}
		}
	}

	for i, delta := range deltas {
		if delta != 0 {
			log.Warning("node %d still has a delta of %d", i, delta)
		}
	}
}
