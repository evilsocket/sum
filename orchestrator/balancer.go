package main

import (
	"fmt"
	pb "github.com/evilsocket/sum/proto"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
)

// I really hate this "success" in the response
func getTheFuckingErrorMessage(err error, response proto.Message) string {
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

func transferOne(fromNode, toNode *NodeInfo, recordId uint64) {
	ctx, cf := newCommContext()
	defer cf()

	record, err := fromNode.Client.ReadRecord(ctx, &pb.ById{Id: recordId})

	if err != nil || !record.Success {
		log.Errorf("Cannot read record %d from node %d: %v", recordId, fromNode.ID, getTheFuckingErrorMessage(err, record))
		return
	}

	if record2, err := fromNode.Client.DeleteRecord(ctx, &pb.ById{Id: recordId}); err != nil || !record2.Success {
		log.Errorf("Cannot delete record %d from node %d: %v", recordId, fromNode.ID, getTheFuckingErrorMessage(err, record2))
		return
	}

	delete(fromNode.RecordIds, recordId)

	newRecord, err := toNode.InternalClient.CreateRecordWithId(ctx, record.Record)

	if err != nil || !newRecord.Success {
		log.Errorf("Unable to create record on node %d: %v", toNode.ID, getTheFuckingErrorMessage(err, newRecord))
		// restore
		if newRecord, err = fromNode.InternalClient.CreateRecordWithId(ctx, record.Record); err != nil || !newRecord.Success {
			log.Errorf("Unable to create record on node %d: %v", fromNode.ID, getTheFuckingErrorMessage(err, newRecord))
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
	var nNodes = uint(len(ms.nodes))
	var maxRecords uint64 = 0
	var maxDelta int64 = 0

	var deltas = make([][]int64, nNodes)
	var id2node = make(map[uint]*NodeInfo, nNodes)
	var id2status = make(map[uint]pb.ServerInfo, nNodes)

	for _, n := range ms.nodes {
		id2node[n.ID] = n
		id2status[n.ID] = n.Status()
	}

	for i := uint(0); i < nNodes; i++ {
		deltas[i] = make([]int64, nNodes)
	}

	for i := uint(0); i < nNodes; i++ {
		for j := i + 1; j < nNodes; j++ {
			deltas[i][j] = int64(id2status[j].Records - id2status[i].Records)
			deltas[j][i] = -deltas[i][j]

			if deltas[i][j] > maxDelta {
				maxDelta = deltas[i][j]
			} else if deltas[j][i] > maxDelta {
				maxDelta = deltas[j][i]
			}
		}
		if id2status[i].Records > maxRecords {
			maxRecords = id2status[i].Records
		}
	}

	// 5% hysteresis
	threshold := int64(maxRecords / 20)

	if maxDelta <= threshold {
		return
	}

	// balance

	for i := uint(0); i < nNodes; i++ {
		for j := i + 1; j < nNodes; j++ {
			records := deltas[i][j]
			if records > threshold || (-records) > threshold {
				src, dst := j, i
				if records < 0 {
					src, dst, records = dst, src, -records
				}
				transfer(id2node[src], id2node[dst], records)
			}
		}
	}
}
