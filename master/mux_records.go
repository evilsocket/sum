package master

import (
	"context"
	"fmt"
	"github.com/evilsocket/islazy/log"
	"github.com/evilsocket/sum/node/storage"
	. "github.com/evilsocket/sum/proto"
	"sort"
	"strings"
)

func (ms *Service) findLessLoadedNode() *NodeInfo {
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

	return targetNode
}

// create a record from the given argument
func (ms *Service) CreateRecord(ctx context.Context, record *Record) (*RecordResponse, error) {
	targetNode := ms.findLessLoadedNode()

	if targetNode == nil {
		return errRecordResponse("No nodes available, try later"), nil
	}

	ms.recordsLock.Lock()
	defer ms.recordsLock.Unlock()

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

// update a record from the given argument
func (ms *Service) UpdateRecord(ctx context.Context, arg *Record) (*RecordResponse, error) {
	ms.recordsLock.RLock()
	defer ms.recordsLock.RUnlock()

	if n, found := ms.recId2node[arg.Id]; !found {
		return errRecordResponse("%v", storage.ErrRecordNotFound), nil
	} else {
		return n.Client.UpdateRecord(ctx, arg)
	}
}

// retrieve a record's content by its id
func (ms *Service) ReadRecord(ctx context.Context, arg *ById) (*RecordResponse, error) {
	ms.recordsLock.RLock()
	defer ms.recordsLock.RUnlock()

	if n, found := ms.recId2node[arg.Id]; !found {
		return errRecordResponse("record %d not found.", arg.Id), nil
	} else {
		return n.Client.ReadRecord(ctx, arg)
	}
}

// list records
func (ms *Service) ListRecords(ctx context.Context, arg *ListRequest) (*RecordListResponse, error) {
	workerInputs := make(map[uint]chan uint64)

	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()
	ms.recordsLock.RLock()
	defer ms.recordsLock.RUnlock()

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
		sortedIds := make([]uint64, 0, len(ms.recId2node))

		for id := range ms.recId2node {
			sortedIds = append(sortedIds, id)
		}

		sort.Slice(sortedIds, func(i, j int) bool { return i < j })

		for _, id := range sortedIds[start:end] {
			n := ms.recId2node[id]
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
					id, n.ID, getErrorMessage(err, resp))
			} else {
				outCh <- resp.Record
			}
		}
	})

	for _, err := range errs {
		log.Error("%s", err)
	}

	for _, res := range results {
		resp.Records = append(resp.Records, res.(*Record))
	}

	sort.Slice(resp.Records, func(i, j int) bool {
		return resp.Records[i].Id < resp.Records[j].Id
	})

	return resp, nil
}

// delete a record by its id
func (ms *Service) DeleteRecord(ctx context.Context, arg *ById) (*RecordResponse, error) {
	ms.recordsLock.Lock()
	defer ms.recordsLock.Unlock()

	if n, found := ms.recId2node[arg.Id]; !found {
		return errRecordResponse("record %d not found.", arg.Id), nil
	} else if resp, err := n.Client.DeleteRecord(ctx, arg); err != nil {
		return resp, err
	} else {
		n.Lock()
		defer n.Unlock()

		delete(n.RecordIds, arg.Id)
		delete(ms.recId2node, arg.Id)
		return &RecordResponse{Success: true}, nil
	}
}

// find records that meet the given requirements
func (ms *Service) FindRecords(ctx context.Context, arg *ByMeta) (*FindResponse, error) {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()
	ms.recordsLock.RLock()
	defer ms.recordsLock.RUnlock()

	results, errs := ms.doParallel(func(n *NodeInfo, resultChannel chan<- interface{}, errorChannel chan<- string) {
		resp, err := n.Client.FindRecords(ctx, arg)
		if err != nil || !resp.Success {
			errorChannel <- getErrorMessage(err, resp)
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

// internal interface ( masters can be slave nodes as well )

// create a record with a given ID
func (ms *Service) CreateRecordWithId(ctx context.Context, in *Record) (*RecordResponse, error) {
	ms.recordsLock.Lock()
	defer ms.recordsLock.Unlock()

	if _, exists := ms.recId2node[in.Id]; exists {
		return errRecordResponse("%v", storage.ErrInvalidID), nil
	}

	n := ms.findLessLoadedNode()
	if n == nil {
		return errRecordResponse("No nodes available, try later"), nil
	}

	n.Lock()
	defer n.Unlock()

	resp, err := n.InternalClient.CreateRecordWithId(ctx, in)
	if err == nil && resp.Success {
		n.RecordIds[in.Id] = true
		n.status.Records++
		ms.recId2node[in.Id] = n
	}
	return resp, err
}

// create multiple records with their preassigned IDs
func (ms *Service) CreateRecordsWithId(ctx context.Context, in *Records) (*RecordResponse, error) {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()
	ms.recordsLock.Lock()
	defer ms.recordsLock.Unlock()

	if len(ms.nodes) == 0 {
		return errRecordResponse("No nodes available, try later"), nil
	}

	for _, r := range in.Records {
		if _, exists := ms.recId2node[r.Id]; exists {
			return errRecordResponse("%v", storage.ErrInvalidID), nil
		}
	}

	perNode := len(in.Records) / len(ms.nodes)
	remainder := len(in.Records) % len(ms.nodes)

	creator := func(n *NodeInfo, records []*Record) error {
		n.Lock()
		defer n.Unlock()

		log.Debug("Master[%s]: creating %d records on node %s", ms.address, len(in.Records), n.Name)

		if resp, err := n.InternalClient.CreateRecordsWithId(ctx, &Records{Records: records}); err != nil || !resp.Success {
			return fmt.Errorf("%s", getErrorMessage(err, resp))
		} else {
			for _, r := range in.Records {
				n.RecordIds[r.Id] = true
				n.status.Records++
				ms.recId2node[r.Id] = n
			}
		}
		return nil
	}

	start := 0
	end := 0
	var successfulNode *NodeInfo
	var lastErr error

	for i, n := range ms.nodes {
		end += perNode
		if i < remainder {
			end++
		}
		if lastErr = creator(n, in.Records[start:end]); lastErr != nil {
			log.Error("Unable to create records on node %d: %v", n.ID, lastErr)
		} else {
			successfulNode = n
			start = end
		}
	}

	if successfulNode == nil {
		return errRecordResponse("Cannot create records on nodes: last error = %v", lastErr), nil
	}

	// last creation failed ( and previous ones potentially )
	if start != end {
		if err := creator(successfulNode, in.Records[start:end]); err != nil {
			// rollback

			arg := &RecordIds{Ids: make([]uint64, 0, len(in.Records))}

			for _, r := range in.Records {
				arg.Ids = append(arg.Ids, r.Id)
			}

			// best effort
			ms.DeleteRecords(ctx, arg)

			return errRecordResponse("Unable to create records on fallback node %d: %v", successfulNode.ID, err), nil
		}
	}

	ms.balance()

	return &RecordResponse{Success: true}, nil
}

func (ms *Service) DeleteRecords(ctx context.Context, in *RecordIds) (*RecordResponse, error) {
	result := &RecordResponse{Success: true}
	node2ids := make(map[*NodeInfo][]uint64)

	ms.recordsLock.Lock()
	defer ms.recordsLock.Unlock()

	for _, id := range in.Ids {
		if n, exists := ms.recId2node[id]; exists {
			node2ids[n] = append(node2ids[n], id)
		}
	}

	for n, ids := range node2ids {
		arg := &RecordIds{Ids: ids}

		func() {
			n.Lock()
			defer n.Unlock()

			resp, err := n.InternalClient.DeleteRecords(ctx, arg)
			if err != nil || !resp.Success {
				result.Success = false
				result.Msg = getErrorMessage(err, resp)
				return
			}

			for _, id := range ids {
				delete(n.RecordIds, id)
				delete(ms.recId2node, id)
				n.status.Records--
			}
		}()

		if !result.Success {
			break
		}
	}

	ms.balance()

	return result, nil
}
