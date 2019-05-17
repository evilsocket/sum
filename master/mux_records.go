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

func (ms *Service) setNextIdIfHigher(newId uint64) {
	ms.idLock.Lock()
	defer ms.idLock.Unlock()
	if ms.nextId <= newId {
		ms.nextId = newId + 1
	}
}

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

	// for targetNode.status.Records++
	targetNode.Lock()
	defer targetNode.Unlock()

	// record.Id = ms.findNextAvailableId()
	// strange, bad but legacy behaviour
	record.Id = ms.nextId

	resp, err := targetNode.InternalClient.CreateRecordWithId(ctx, record)

	if err == nil && resp.Success {
		ms.nextId++
		targetNode.status.Records++
		targetNode.status.NextRecordId = ms.nextId
	}

	return resp, err
}

// update a record from the given argument
func (ms *Service) UpdateRecord(_ context.Context, arg *Record) (*RecordResponse, error) {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	ctx, cf := newCommContext()
	defer cf()

	results, errs := ms.doParallel(func(node *NodeInfo, resultChannel chan<- interface{}, errorChannel chan<- string) {
		resp, err := node.Client.UpdateRecord(ctx, arg)
		if err != nil || !resp.Success {
			msg := getErrorMessage(err, resp)
			if msg != storage.ErrRecordNotFound.Error() {
				errorChannel <- fmt.Sprintf("node %d: %v", node.ID, msg)
			}
		} else {
			cf() // cancel other queries
			resultChannel <- true
		}
	})

	switch len(results) {
	case 0:
		if len(errs) == 0 {
			return errRecordResponse("%v", storage.ErrRecordNotFound), nil
		}
		return errRecordResponse("No node was able to satisfy your request: [%s]", strings.Join(errs, ", ")), nil
	default:
		log.Warning("Got multiple results when only one was expected: %v", results)
		fallthrough
	case 1:
		return &RecordResponse{Success: true}, nil
	}
}

// retrieve a record's content by its id
func (ms *Service) ReadRecord(_ context.Context, arg *ById) (*RecordResponse, error) {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	notFoundError := fmt.Sprintf("record %d not found.", arg.Id)

	ctx, cf := newCommContext()
	defer cf()

	results, errs := ms.doParallel(func(node *NodeInfo, resultChannel chan<- interface{}, errorChannel chan<- string) {
		resp, err := node.Client.ReadRecord(ctx, arg)
		if err != nil || !resp.Success {
			msg := getErrorMessage(err, resp)
			if msg != notFoundError {
				errorChannel <- fmt.Sprintf("node %d: %v", node.ID, msg)
			}
		} else {
			cf() // cancel other queries
			resultChannel <- resp.Record
		}
	})

	switch len(results) {
	case 0:
		if len(errs) == 0 {
			return errRecordResponse("%s", notFoundError), nil
		}
		return errRecordResponse("No node was able to satisfy your request: [%s]", strings.Join(errs, ", ")), nil
	default:
		log.Warning("Got multiple results when only one was expected: %v", results)
		fallthrough
	case 1:
		return &RecordResponse{Success: true, Record: results[0].(*Record)}, nil
	}
}

// list records
func (ms *Service) ListRecords(ctx context.Context, arg *ListRequest) (*RecordListResponse, error) {

	if arg.PerPage == 0 {
		return nil, fmt.Errorf("invalid arguments")
	}

	if arg.Page == 0 {
		arg.Page = 1
	}

	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	var orderedNodes = make([]*NodeInfo, 0, len(ms.nodes))
	var total uint64 = 0

	for _, n := range ms.nodes {
		n.RLock()
		defer n.RUnlock()
		orderedNodes = append(orderedNodes, n)
		total += n.status.Records
	}

	sort.Slice(orderedNodes, func(i, j int) bool {
		return orderedNodes[i].ID < orderedNodes[j].ID
	})

	pages := total / arg.PerPage
	if total%arg.PerPage > 0 {
		pages++
	}

	start := arg.PerPage * (arg.Page - 1)
	end := start + arg.PerPage
	cursor := uint64(0)
	firstNodeIndex := -1
	lastNodeIndex := -1
	lastNodeId := uint(1)
	lastNodeRecords := uint64(0)

	if end == start {
		return &RecordListResponse{Records: []*Record{}, Pages: pages, Total: total}, nil
	}

	for i, n := range orderedNodes {
		lastNodeId = n.ID

		if cursor <= start && cursor+n.status.Records > start {
			firstNodeIndex = i
		}

		if cursor < end && cursor+n.status.Records >= end {
			lastNodeIndex = i
			lastNodeRecords = end - cursor
			break
		}

		cursor = cursor + n.status.Records
	}

	if firstNodeIndex == -1 || lastNodeIndex == -1 {
		return &RecordListResponse{Records: []*Record{}, Pages: pages, Total: total}, nil
	}

	toQueryNodes := orderedNodes[firstNodeIndex : lastNodeIndex+1]

	results, errs := doParallel(toQueryNodes, func(node *NodeInfo, resultChannel chan<- interface{}, errorChannel chan<- string) {
		arg := &ListRequest{Page: 1}

		if node.ID == lastNodeId {
			arg.PerPage = lastNodeRecords
		} else {
			// this size will never exceed the original request,
			// which we assume to be reasonable
			arg.PerPage = node.status.Records
		}

		resp, err := node.Client.ListRecords(ctx, arg)
		if err != nil {
			errorChannel <- err.Error()
		} else {
			resultChannel <- resp.Records
		}
	})

	if len(errs) > 0 {
		return nil, fmt.Errorf("unable to communicate with nodes: [%s]", strings.Join(errs, ", "))
	}

	records := make([]*Record, 0, arg.PerPage)
	for _, ary := range results {
		for _, r := range ary.([]*Record) {
			records = append(records, r)
		}
	}

	return &RecordListResponse{Total: total, Pages: pages, Records: records}, nil
}

// delete a record by its id
func (ms *Service) DeleteRecord(_ context.Context, arg *ById) (*RecordResponse, error) {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	notFoundError := fmt.Sprintf("record %d not found.", arg.Id)

	ctx, cf := newCommContext()
	defer cf()

	results, errs := ms.doParallel(func(node *NodeInfo, resultChannel chan<- interface{}, errorChannel chan<- string) {
		node.Lock()
		defer node.Unlock()

		resp, err := node.Client.DeleteRecord(ctx, arg)
		if err != nil || !resp.Success {
			msg := getErrorMessage(err, resp)
			if msg != notFoundError {
				errorChannel <- fmt.Sprintf("node %d: %v", node.ID, msg)
			}
		} else {
			cf() // cancel other queries
			node.status.Records--
			resultChannel <- true
		}
	})

	switch len(results) {
	case 0:
		if len(errs) == 0 {
			return errRecordResponse("%v", notFoundError), nil
		}
		return errRecordResponse("No node was able to satisfy your request: [%s]", strings.Join(errs, ", ")), nil
	default:
		log.Warning("Got %d results when only one was expected", len(results))
		fallthrough
	case 1:
		return &RecordResponse{Success: true}, nil
	}
}

// find records that meet the given requirements
func (ms *Service) FindRecords(ctx context.Context, arg *ByMeta) (*FindResponse, error) {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

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
func (ms *Service) CreateRecordWithId(_ context.Context, in *Record) (*RecordResponse, error) {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	// TODO: fix double read lock
	n := ms.findLessLoadedNode()
	if n == nil {
		return errRecordResponse("No nodes available, try later"), nil
	}

	// must query the record from nodes to check its existence

	ctx, cf := newCommContext()
	defer cf()

	results, _ := ms.doParallel(func(node *NodeInfo, resultChannel chan<- interface{}, errorChannel chan<- string) {
		resp, err := node.Client.ReadRecord(ctx, &ById{Id: in.Id})
		if err == nil && resp.Success {
			cf() // cancel other queries
			resultChannel <- true
		}
	})

	if len(results) > 0 {
		return errRecordResponse("%v", storage.ErrInvalidID), nil
	}

	n.Lock()
	defer n.Unlock()

	resp, err := n.InternalClient.CreateRecordWithId(ctx, in)
	if err == nil && resp.Success {
		n.status.Records++
		if n.status.NextRecordId <= in.Id {
			n.status.NextRecordId = in.Id + 1
		}
		ms.idLock.Lock()
		defer ms.idLock.Unlock()
		if ms.nextId < n.status.NextRecordId {
			ms.nextId = n.status.NextRecordId
		}
	}
	return resp, err
}

// create multiple records with their preassigned IDs
func (ms *Service) CreateRecordsWithId(ctx context.Context, in *Records) (*RecordResponse, error) {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	if len(ms.nodes) == 0 {
		return errRecordResponse("No nodes available, try later"), nil
	}

	perNode := len(in.Records) / len(ms.nodes)
	remainder := len(in.Records) % len(ms.nodes)

	creator := func(n *NodeInfo, records []*Record) error {
		maxId := uint64(0)
		for _, r := range records {
			if r.Id > maxId {
				maxId = r.Id
			}
		}

		n.Lock()
		defer n.Unlock()

		log.Debug("Master[%s]: creating %d records on node %s", ms.address, len(records), n.Name)

		if resp, err := n.InternalClient.CreateRecordsWithId(ctx, &Records{Records: records}); err != nil || !resp.Success {
			return fmt.Errorf("%s", getErrorMessage(err, resp))
		} else {
			n.status.Records += uint64(len(records))
			if n.status.NextRecordId <= maxId {
				n.status.NextRecordId = maxId + 1
				ms.setNextIdIfHigher(maxId + 1)
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

	oldTotal := uint64(0)
	newTotal := uint64(0)

	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	for _, n := range ms.nodes {
		oldTotal += n.Status().Records
	}

	ms.doParallel(func(node *NodeInfo, resultChannel chan<- interface{}, errorChannel chan<- string) {
		node.InternalClient.DeleteRecords(ctx, in)
		node.UpdateStatus()

		ms.setNextIdIfHigher(node.Status().NextRecordId)
	})

	ms.balance()

	for _, n := range ms.nodes {
		newTotal += n.Status().Records
	}

	deleted := oldTotal - newTotal
	if uint64(len(in.Ids)) != deleted {
		return errRecordResponse("deleted %d records out of %d", deleted, len(in.Ids)), nil
	}

	return &RecordResponse{Success: true}, nil
}
