package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	pb "github.com/evilsocket/sum/proto"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"strings"
	"sync"
)

// bring peace among nodes that contest a certain record id.
// this may happen when a new node is added, with already loaded records.
// peacemaker will find collision and apply a fix:
//  - leave the record only on one node when they are the same
//  - change the record id when different
func (ms *MuxService) solveAllConflictsInTheWorld() error {
	// maps record's ID to its contending nodes
	var conflicts = make(map[uint64][]*NodeInfo)

	if len(ms.nodes) < 2 {
		return nil
	}

	for _, n := range ms.nodes {
		n.Lock()
		defer n.Unlock() // ensure persistent status across the whole function
		for rId := range n.RecordIds {
			conflicts[rId] = append(conflicts[rId], n)
		}
	}

	// delete peaceful records
	for rId, nodes := range conflicts {
		if len(nodes) < 2 {
			delete(conflicts, rId)
		}
	}

	var overallErrors []error

	for rId, nodes := range conflicts {
		if err := ms.solveConflict(rId, nodes); err != nil {
			overallErrors = append(overallErrors, err)
		}
	}

	switch len(overallErrors) {
	case 0:
		return nil
	case 1:
		return fmt.Errorf("error in solving conflict: %v", overallErrors[0])
	default:
		return fmt.Errorf("multiple errors in solving conflicts: %v", errorsToString(overallErrors))
	}
}

func (ms *MuxService) solveConflict(rId uint64, nodes []*NodeInfo) error {
	var mapLock sync.Mutex
	var records = make(map[*NodeInfo]*pb.Record)

	// fetch records from all contending nodes

	_, errs := doParallel(nodes, func(n *NodeInfo, _ chan<- interface{}, errorChannel chan<- string) {
		if resp, err := n.Client.ReadRecord(context.Background(), &pb.ById{Id: rId}); err != nil {
			errorChannel <- fmt.Sprintf("Cannot retrieve record %d from node %d: %v", rId, n.ID, err)
		} else {
			mapLock.Lock()
			defer mapLock.Unlock()
			records[n] = resp.Record
		}
	})

	if len(errs) > 0 {
		return fmt.Errorf("unable to fetch records from other nodes: [%s]", strings.Join(errs, ", "))
	}

	// group nodes by record hash

	var recordHash2nodes = make(map[string][]*NodeInfo)

	for n, r := range records {
		if data, err := proto.Marshal(r); err != nil {
			return fmt.Errorf("unable to marshal record %d: %v", r.Id, err)
		} else {
			hashb := sha256.Sum256(data)
			hash := hex.EncodeToString(hashb[:])
			recordHash2nodes[hash] = append(recordHash2nodes[hash], n)
		}
	}

	// for each group keep only one copy of the record

	var overallErrors []error
	firstTime := true

	for _, nodes := range recordHash2nodes {
		// the lucky one who keep the record
		n := nodes[0]

		// ensure a consistent state in case of failure
		if firstTime {
			ms.recId2node[rId] = n
		}

		if err := ms.deleteRecordFromNodes(rId, nodes[1:]); err != nil {
			overallErrors = append(overallErrors, err)
			continue
		}

		if firstTime {
			firstTime = false
			continue
		}

		// change record Id
		// this happens when records with the same Id are different across nodes

		if err := ms.changeRecordIdOnNode(records[n], n); err != nil {
			overallErrors = append(overallErrors, err)
		}
	}

	switch len(overallErrors) {
	case 0:
		return nil
	case 1:
		return fmt.Errorf("error during conflict resolution: %v", overallErrors[0])
	default:
		return fmt.Errorf("multiple errors during conflict resolution: [%s]", errorsToString(overallErrors))
	}
}

func (ms *MuxService) deleteRecordFromNodes(rId uint64, nodes []*NodeInfo) error {
	for _, n := range nodes {
		if resp, err := n.Client.DeleteRecord(context.Background(), &pb.ById{Id: rId}); err != nil || !resp.Success {
			return fmt.Errorf("unable to delete record %d on node %d: %v", rId, n.ID, getTheFuckingErrorMessage(err, resp))
		}
		delete(n.RecordIds, rId)
	}

	return nil
}

func (ms *MuxService) changeRecordIdOnNode(r *pb.Record, n *NodeInfo) error {
	rId := r.Id
	r.Id = ms.findNextAvailableId()

	if resp, err := n.Client.DeleteRecord(context.Background(), &pb.ById{Id: rId}); err != nil || !resp.Success {
		return fmt.Errorf("unable to delete record %d on node %d: %v", rId, n.ID, getTheFuckingErrorMessage(err, resp))
	}

	delete(n.RecordIds, rId)

	if resp, err := n.InternalClient.CreateRecordWithId(context.Background(), r); err != nil || !resp.Success {
		return fmt.Errorf("unable to create record %d on node %d: %v", r.Id, n.ID, getTheFuckingErrorMessage(err, resp))
	}

	log.Infof("Record %d has been changed to record %d on node %d", rId, r.Id, n.ID)

	n.RecordIds[r.Id] = true
	ms.recId2node[r.Id] = n

	return nil
}

func errorsToString(ary []error) string {
	errs := make([]string, len(ary))
	for i, e := range ary {
		errs[i] = e.Error()
	}
	return strings.Join(errs, ", ")
}
