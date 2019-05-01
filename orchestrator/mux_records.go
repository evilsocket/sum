package main

import (
	"context"
	"fmt"
	. "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/storage"
	log "github.com/sirupsen/logrus"
	"sort"
	"strings"
)

func (ms *MuxService) CreateRecord(ctx context.Context, record *Record) (*RecordResponse, error) {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	var lowestRecords *uint64
	var targetNode *NodeInfo

	for _, n := range ms.nodes {
		st := n.Status()
		if lowestRecords == nil || st.Records < *lowestRecords {
			lowestRecords = &st.Records
			targetNode = n
		}
	}

	if targetNode == nil {
		return errRecordResponse("No nodes available, try later"), nil
	}

	// for targetNode.status.Records++
	targetNode.Lock()
	defer targetNode.Unlock()

	// record.Id = ms.findNextAvailableId()
	// strange, bad but legacy behaviour
	record.Id = ms.nextId
	if _, exists := ms.recId2node[record.Id]; exists {
		return errRecordResponse("%v", storage.ErrInvalidID), nil
	}

	resp, err := targetNode.InternalClient.CreateRecordWithId(ctx, record)

	if err == nil && resp.Success {
		ms.nextId++
		targetNode.RecordIds[record.Id] = true
		ms.recId2node[record.Id] = targetNode
		targetNode.status.Records++
	}

	return resp, err
}

func (ms *MuxService) UpdateRecord(ctx context.Context, arg *Record) (*RecordResponse, error) {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	if n, found := ms.recId2node[arg.Id]; !found {
		return errRecordResponse("%v", storage.ErrRecordNotFound), nil
	} else {
		return n.Client.UpdateRecord(ctx, arg)
	}
}

func (ms *MuxService) ReadRecord(ctx context.Context, arg *ById) (*RecordResponse, error) {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	if n, found := ms.recId2node[arg.Id]; !found {
		return errRecordResponse("record %d not found.", arg.Id), nil
	} else {
		return n.Client.ReadRecord(ctx, arg)
	}
}

func (ms *MuxService) ListRecords(ctx context.Context, arg *ListRequest) (*RecordListResponse, error) {
	workerInputs := make(map[uint]chan uint64)

	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	for _, n := range ms.nodes {
		n.RLock()
		defer n.RUnlock() // defer: ensure consistent data across this whole function
		workerInputs[n.ID] = make(chan uint64, 1)
	}

	total := uint64(len(ms.recId2node))
	pages := total / arg.PerPage
	if total%pages > 0 {
		pages++
	}
	start := arg.PerPage * (arg.Page - 1)
	end := start + arg.PerPage
	if end > total {
		end = total
	}
	if start > end {
		start = end
	}
	resp := &RecordListResponse{Total: total, Pages: pages, Records: make([]*Record, 0, end-start)}

	if total == 0 || end == start {
		return resp, nil
	}

	go func() {
		for id, n := range ms.recId2node {
			workerInputs[n.ID] <- id
		}
		for _, ch := range workerInputs {
			close(ch)
		}
	}()

	//NB: can be improved by spawning more workers per node, each one with a different connection
	results, errs := ms.doParallel(func(n *NodeInfo, outCh chan<- interface{}, errCh chan<- string) {
		for id := range workerInputs[n.ID] {
			resp, err := n.Client.ReadRecord(ctx, &ById{Id: id})
			if err != nil || !resp.Success {
				errCh <- fmt.Sprintf("Unable to read record %d on node %d: %v",
					id, n.ID, getTheFuckingErrorMessage(err, resp))
			} else {
				outCh <- resp.Record
			}
		}
	})

	for _, err := range errs {
		log.Errorf("%s", err)
	}

	for _, res := range results {
		resp.Records = append(resp.Records, res.(*Record))
	}

	sort.Slice(resp.Records, func(i, j int) bool {
		return resp.Records[i].Id < resp.Records[j].Id
	})

	resp.Records = resp.Records[start:end]

	return resp, nil
}

func (ms *MuxService) DeleteRecord(ctx context.Context, arg *ById) (*RecordResponse, error) {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	if n, found := ms.recId2node[arg.Id]; !found {
		return errRecordResponse("record %d not found.", arg.Id), nil
	} else if resp, err := n.Client.DeleteRecord(ctx, arg); err != nil {
		return resp, err
	} else {
		delete(n.RecordIds, arg.Id)
		delete(ms.recId2node, arg.Id)
		return &RecordResponse{Success: true}, nil
	}
}

func (ms *MuxService) FindRecords(ctx context.Context, arg *ByMeta) (*FindResponse, error) {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	results, errs := ms.doParallel(func(n *NodeInfo, resultChannel chan<- interface{}, errorChannel chan<- string) {
		resp, err := n.Client.FindRecords(ctx, arg)
		if err != nil || !resp.Success {
			errorChannel <- getTheFuckingErrorMessage(err, resp)
		} else {
			for _, r := range resp.Records {
				resultChannel <- r
			}
		}
	})

	if len(errs) > 0 {
		return errFindResponse("Errors from nodes: [%s]", strings.Join(errs, ", ")), nil
	}

	records := make([]*Record, 0, len(results))
	for _, r := range results {
		records = append(records, r.(*Record))
	}

	return &FindResponse{Success: true, Records: records}, nil
}
