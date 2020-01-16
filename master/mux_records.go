package master

import (
	"context"
	"fmt"
	"github.com/evilsocket/islazy/log"
	"github.com/evilsocket/sum/node/storage"
	. "github.com/evilsocket/sum/proto"
	"math/big"
	"sort"
	"strconv"
	"strings"
)

func (ms *Service) setNextIdIfHigher(newId uint64) {
	ms.idLock.Lock()
	defer ms.idLock.Unlock()
	if ms.nextId <= newId {
		ms.nextId = newId + 1
	}
}

func (ms *Service) _findLessLoadedNode() *NodeInfo {
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

func (ms *Service) findLessLoadedNode() *NodeInfo {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	return ms._findLessLoadedNode()
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

	ms.idLock.Lock()
	defer ms.idLock.Unlock()

	record.Id = ms.nextId

	resp, err := targetNode.InternalClient.CreateRecordWithId(ctx, record)

	if err == nil && resp.Success {
		ms.nextId++
		targetNode.status.Records++
		targetNode.status.NextRecordId = ms.nextId
	}

	return resp, err
}

// CreateRecords creates and stores a series of new *pb.Record object.
func (ms *Service) CreateRecords(ctx context.Context, records *Records) (*RecordResponse, error) {
	targetNode := ms.findLessLoadedNode()
	if targetNode == nil {
		return errRecordResponse("No nodes available, try later"), nil
	}

	// for targetNode.status.Records++
	targetNode.Lock()
	defer targetNode.Unlock()

	ms.idLock.Lock()
	defer ms.idLock.Unlock()

	// set the identifiers
	for _, record := range records.Records {
		record.Id = ms.nextId
		ms.nextId++
	}

	resp, err := targetNode.InternalClient.CreateRecordsWithId(ctx, records)
	if err == nil && resp.Success {
		targetNode.status.Records += uint64(len(records.Records))
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
		log.Warning("Got %d results when only one was expected: %v", len(results), results)
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
		log.Warning("Got %d results when only one was expected: %v", len(results), results)
		fallthrough
	case 1:
		return &RecordResponse{Success: true, Record: results[0].(*Record)}, nil
	}
}

// list records
func (ms *Service) ListRecords(ctx context.Context, arg *ListRequest) (*RecordListResponse, error) {
	if arg.Page < 1 {
		arg.Page = 1
	}

	if arg.PerPage < 1 {
		arg.PerPage = 1
	}

	fetcher := NewRecordFetcher()

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

	for _, n := range orderedNodes {
		nodeEnd := cursor + n.status.Records
		records := n.status.Records

		// page start at this node
		if cursor <= start && nodeEnd > start {
			offset := start - cursor
			records -= offset

			// page also end in this node
			if end <= nodeEnd {
				records = arg.PerPage
			}

			if offset == 0 {
				// page start with this node, get first page, same perPage
				fetcher.Fetch(n, 1, records)
			} else if end >= nodeEnd && records < offset {
				// optimization using overflowed page
				fetcher.Fetch(n, 2, offset)
			} else {
				// find the GCD between offset and records to
				// skip offset and fetch exactly the needed records

				bigOffset := big.NewInt(0).SetUint64(offset)
				bigRecords := big.NewInt(0).SetUint64(records)
				gcd := big.NewInt(0).GCD(nil, nil, bigOffset, bigRecords).Uint64()

				startPage := offset/gcd + 1
				nPages := records / gcd
				for i := uint64(0); i < nPages; i++ {
					fetcher.Fetch(n, startPage+i, gcd)
				}
			}

			if end <= nodeEnd {
				break
			}
		} else if cursor < end && nodeEnd > end {
			// chunk end at this node
			fetcher.Fetch(n, 1, end-cursor)
			break
		} else if start < cursor && end > nodeEnd {
			fetcher.Fetch(n, 1, records)
		}

		cursor = cursor + n.status.Records
	}

	fetcher.Wait()

	if len(fetcher.Errs) > 0 {
		log.Warning("unable to communicate with nodes: [%s]", strings.Join(fetcher.Errs, ", "))
	}

	return &RecordListResponse{Total: total, Pages: pages, Records: fetcher.Records}, nil
}

// delete a record by its id
func (ms *Service) DeleteRecord(_ context.Context, arg *ById) (*RecordResponse, error) {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	notFoundError := fmt.Sprintf("record %d not found.", arg.Id)

	ctx, cf := context.WithCancel(context.Background())
	if !commContextIsCancellable {
		cf = func() {}
	} else {
		defer cf()
	}

	results, errs := ms.doParallel(func(node *NodeInfo, resultChannel chan<- interface{}, errorChannel chan<- string) {
		node.Lock()
		defer node.Unlock()

		ctx, _ := context.WithTimeout(ctx, timeout)

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
	notIndexedErrmsg := fmt.Sprintf("meta %v not indexed.", arg.Meta)

	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	results, errs := ms.doParallel(func(n *NodeInfo, resultChannel chan<- interface{}, errorChannel chan<- string) {
		resp, err := n.Client.FindRecords(ctx, arg)
		if err != nil || !resp.Success {
			msg := getErrorMessage(err, resp)
			if msg != notIndexedErrmsg {
				errorChannel <- msg
			}
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

	n := ms._findLessLoadedNode()
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

		ctx, cf := newCommContext()
		defer cf()

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
			ms._deleteRecords(ctx, arg)

			return errRecordResponse("Unable to create records on fallback node %d: %v", successfulNode.ID, err), nil
		}
	}

	ms.balance()

	return &RecordResponse{Success: true}, nil
}

func (ms *Service) _deleteRecords(ctx context.Context, in *RecordIds) (*RecordResponse, error) {
	ctx, cf := newCommContext()
	defer cf()

	result, errs := ms.doParallel(func(node *NodeInfo, resultChannel chan<- interface{}, errorChannel chan<- string) {
		if resp, err := node.InternalClient.DeleteRecords(ctx, in); err != nil {
			errorChannel <- fmt.Sprintf("comunication error with node %v: %v", node.Name, err)
			cf()
		} else if numDeleted, err := strconv.ParseUint(resp.Msg, 10, 64); err != nil {
			errorChannel <- fmt.Sprintf("unable to parse node '%v' response '%v' as uint: %v", node.Name, resp.Msg, err)
			cf()
		} else if numDeleted != 0 {
			node.UpdateStatus()

			ms.setNextIdIfHigher(node.Status().NextRecordId)
			resultChannel <- numDeleted
		}
	})

	ms.balance()

	if len(errs) > 0 {
		return errRecordResponse("errors from nodes: [%s]", strings.Join(errs, ",")), nil
	}

	deleted := uint64(0)
	for _, res := range result {
		deleted += res.(uint64)
	}

	ok := uint64(len(in.Ids)) == deleted

	return &RecordResponse{Success: ok, Msg: fmt.Sprintf("%d", deleted)}, nil
}

func (ms *Service) DeleteRecords(ctx context.Context, in *RecordIds) (*RecordResponse, error) {
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()

	return ms._deleteRecords(ctx, in)
}
