package main

import (
	"fmt"
	pb "github.com/evilsocket/sum/proto"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
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

// transfer a record from one node to another
func transferOne(fromNode, toNode *NodeInfo, recordId uint64) {
	ctx, cf := newCommContext()
	defer cf()

	record, err := fromNode.Client.ReadRecord(ctx, &pb.ById{Id: recordId})

	if err != nil || !record.Success {
		log.Errorf("Cannot read record %d from node %d: %v", recordId, fromNode.ID, getErrorMessage(err, record))
		return
	}

	if record2, err := fromNode.Client.DeleteRecord(ctx, &pb.ById{Id: recordId}); err != nil || !record2.Success {
		log.Errorf("Cannot delete record %d from node %d: %v", recordId, fromNode.ID, getErrorMessage(err, record2))
		return
	}

	delete(fromNode.RecordIds, recordId)

	newRecord, err := toNode.InternalClient.CreateRecordWithId(ctx, record.Record)

	if err != nil || !newRecord.Success {
		log.Errorf("Unable to create record on node %d: %v", toNode.ID, getErrorMessage(err, newRecord))
		// restore
		if newRecord, err = fromNode.InternalClient.CreateRecordWithId(ctx, record.Record); err != nil || !newRecord.Success {
			log.Errorf("Unable to create record on node %d: %v", fromNode.ID, getErrorMessage(err, newRecord))
			log.Errorf("Record %d lost ( %s )", record.Record.Id, record.Record.Meta)
		} else {
			log.Infof("Record %d restored on node %d", recordId, fromNode.ID)
			fromNode.RecordIds[recordId] = true
		}
	} else {
		toNode.RecordIds[recordId] = true
	}
}

// transfer nRecords from a node to another
func transfer(fromNode, toNode *NodeInfo, nRecords int64) {
	i := int64(0)

	fromNode.Lock()
	toNode.Lock()
	defer fromNode.Unlock()
	defer toNode.Unlock()

	for id := range fromNode.RecordIds {
		transferOne(fromNode, toNode, id)
		i++
		if i >= nRecords {
			break
		}
	}
}

// balance the load among nodes
func (ms *MuxService) balance() {
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
			transfer(ms.nodes[j], ms.nodes[i], nRecords)
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
			log.Warnf("Node %d still have a delta of %d", i, delta)
		}
	}
}
