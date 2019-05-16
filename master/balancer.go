package master

import (
	"fmt"

	"github.com/golang/protobuf/proto"

	pb "github.com/evilsocket/sum/proto"

	"github.com/evilsocket/islazy/log"
)

// Get the error message from either a GRPC error or a application-level one
func getErrorMessage(err error, response proto.Message) string {
	if err != nil {
		return err.Error()
	}

	var success bool
	var msg string

	switch response.(type) {
	case *pb.RecordResponse:
		success = response.(*pb.RecordResponse).Success
		msg = response.(*pb.RecordResponse).Msg
	case *pb.OracleResponse:
		success = response.(*pb.OracleResponse).Success
		msg = response.(*pb.OracleResponse).Msg
	case *pb.CallResponse:
		success = response.(*pb.CallResponse).Success
		msg = response.(*pb.CallResponse).Msg
	default:
		panic(fmt.Sprintf("unsupported message %T: %v", response, response))
	}

	if !success {
		return msg
	}

	panic("no errors dude")
}

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
		toNode.RecordIds[r.Id] = true
		ms.recId2node[r.Id] = toNode
		delReq.Ids = append(delReq.Ids, r.Id)
	}

	resp1, err := fromNode.InternalClient.DeleteRecords(ctx, delReq)

	if err != nil || !resp1.Success {
		log.Error("Unable to delete records from node %d: %v", fromNode.ID, getErrorMessage(err, resp1))
		// keep going anyway
	}

	log.Debug("Master[%s]: deleted %d records from node %s", ms.address, len(delReq.Ids), fromNode.Name)

	fromNode.status.Records -= uint64(nRecords)

	for _, r := range list.Records {
		delete(fromNode.RecordIds, r.Id)
	}
}

// balance the load among nodes
func (ms *Service) balance() {
	log.Debug("Master[%s]: balancing...", ms.address)
	var totRecords = uint64(len(ms.recId2node))
	var nNodes = len(ms.nodes)

	if totRecords == 0 || nNodes == 0 {
		return
	}

	var targetRecordsPerNode = totRecords / uint64(nNodes)
	var reminder = int(totRecords % uint64(nNodes))
	// target amount of records
	var targets = make([]uint64, nNodes)
	// nodes that shall enter in the node
	var deltas = make([]int64, nNodes)
	var needsBalancing = false

	for i, n := range ms.nodes {
		targets[i] = targetRecordsPerNode
		if i < reminder {
			targets[i]++
		}
		deltas[i] = int64(targets[i]) - int64(n.status.Records)

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
