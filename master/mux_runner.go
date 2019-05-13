package master

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/evilsocket/sum/node/wrapper"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/evilsocket/sum/node/service"
	. "github.com/evilsocket/sum/proto"

	"github.com/evilsocket/islazy/log"
)

// run an oracle with the given arguments and get its results back
// in this implementation the original oracle is patched and sent down
// to the nodes. It is then run in parallel and its results merged together.
// Because of this merging, if the oracle returns a scalar a merging function is needed.
// To declare a merging function just declare a function whose name begin with
// "merge". Please remember that the first function shall be the oracle.
func (ms *Service) Run(ctx context.Context, arg *Call) (*CallResponse, error) {

	// NB: always keep this order of locking
	ms.nodesLock.RLock()
	defer ms.nodesLock.RUnlock()
	ms.recordsLock.RLock()
	defer ms.recordsLock.RUnlock()
	ms.cageLock.RLock()
	defer ms.cageLock.RUnlock()

	raccoon, found := ms.raccoons[arg.OracleId]
	if !found {
		return errCallResponse("oracle %d not found.", arg.OracleId), nil
	}

	// 1. Find the record the oracle is working on

	resolvedRecords := make([]*Record, len(raccoon.parameters)) // fill with nil

	for i, a := range arg.Args {
		if !raccoon.IsParameterPositionARecordLookup(i) {
			continue
		}

		recId, err := strconv.ParseUint(a, 10, 64)
		if err != nil {
			return errCallResponse("Unable to parse record id form parameter #%d: %v", i, err), nil
		}
		node, found := ms.recId2node[recId]
		if !found {
			//FIXME: we shell make records.Find(...) return `null` when this happens
			return errCallResponse("Record %d not found", recId), nil
		}
		record, err := node.Client.ReadRecord(ctx, &ById{Id: recId})
		if err != nil || !record.Success {
			return errCallResponse("Unable to retrieve record %d form node %d: %v",
				recId, node.ID, getErrorMessage(err, record)), nil
		}
		resolvedRecords[i] = record.Record
	}

	// 2. substitute all the calls to records.Find(...) with their resolved record

	newCode, err := raccoon.PatchCode(resolvedRecords)
	if err != nil {
		return errCallResponse("Unable to patch JS code: %v", err), nil
	}

	// 3. create the modified oracle on all nodes
	node2oracleId := make(map[*NodeInfo]uint64)
	mapLock := sync.Mutex{}
	newOracle := &Oracle{Code: newCode, Name: raccoon.Name}

	// cleanup created oracles
	defer func() {
		for n, oId := range node2oracleId {
			resp, err := n.Client.DeleteOracle(ctx, &ById{Id: oId})
			if err != nil || !resp.Success {
				log.Warning("Unable to delete temporary oracle %d on node %d: %v",
					oId, n.ID, getErrorMessage(err, resp))
			}
		}
	}()

	results, errs := ms.doParallel(func(n *NodeInfo, okChan chan<- interface{}, errChan chan<- string) {
		resp, err := n.Client.CreateOracle(ctx, newOracle)
		if err != nil || !resp.Success {
			errChan <- getErrorMessage(err, resp)
			return
		}
		oId, err := strconv.ParseUint(resp.Msg, 10, 64)
		if err != nil {
			errChan <- fmt.Sprintf("unable to parse oracleId string '%s': %v", resp.Msg, err)
			return
		}
		func() {
			mapLock.Lock()
			defer mapLock.Unlock()
			node2oracleId[n] = oId
		}()
		resp1, err := n.Client.Run(ctx, &Call{OracleId: oId, Args: arg.Args})
		if err != nil || !resp1.Success {
			errChan <- getErrorMessage(err, resp1)
			return
		}
		if resp1.Data.Compressed {
			if r, err := gzip.NewReader(bytes.NewReader(resp1.Data.Payload)); err != nil {
				errChan <- err.Error()
				return
			} else if resp1.Data.Payload, err = ioutil.ReadAll(r); err != nil {
				errChan <- err.Error()
				return
			}
		}
		var res interface{}
		if err = json.Unmarshal(resp1.Data.Payload, &res); err != nil {
			errChan <- err.Error()
			return
		}
		okChan <- res
	})

	if len(errs) > 0 {
		return errCallResponse("Errors from nodes: [%s]", strings.Join(errs, ", ")), nil
	}

	if mergedResults, err := ms.merge(raccoon, results); err != nil {
		return errCallResponse("Unable to merge results from nodes: %v", err), nil
	} else if raw, err := json.Marshal(mergedResults); err != nil {
		return errCallResponse("Unable to marshal result: %v", err), nil
	} else {
		return &CallResponse{Success: true, Msg: "", Data: service.BuildPayload(raw)}, nil
	}
}

// merge results together
func (ms *Service) merge(raccoon *astRaccoon, results []interface{}) (interface{}, error) {
	if raccoon.MergerFunction == nil {
		return ms.defaultMerger(results)
	}
	vm := ms.vmPool.Get()
	defer vm.Release()

	mf := raccoon.MergerFunction
	ctx := wrapper.NewContext()

	if err := vm.Set(mf.ParameterList.List[0].Name, results); err != nil {
		return errCallResponse("Unable to set parameter variable '%s': %v", mf.ParameterList.List[0].Name, err), nil
	} else if err := vm.Set("ctx", ctx); err != nil {
		return errCallResponse("Unable to set parameter variable '%s': %v", "ctx", err), nil
	}

	// I've tried with the compiled version but didn't work ^^"
	code := fmt.Sprintf("%s\n%s(%s)",
		raccoon.src, raccoon.MergerFunction.Name.Name, raccoon.MergerFunction.ParameterList.List[0].Name)

	ret, err := vm.Run(code)

	if err != nil {
		return errCallResponse("Unable to run merger function: %v", err), nil
	} else if ctx.IsError() {
		// same goes for errors triggered within the oracle
		return errCallResponse("Merger function failed: %v", ctx.Message()), nil
	} else if mergedResults, err := ret.Export(); err != nil {
		// or if we can't export its return value
		return errCallResponse("Couldn't deserialize returned object from merger: %v", err), nil
	} else {
		return mergedResults, nil
	}
}

// default merger for maps and arrays
func (_ *Service) defaultMerger(results []interface{}) (mergedResults interface{}, _ error) {
	var resultType *reflect.Type

	mergedResults = nil

	for _, res := range results {
		t := reflect.TypeOf(res)
		if resultType == nil {
			resultType = &t
		} else if *resultType != t {
			return nil, fmt.Errorf("heterogeneous results: prior results had type %v, this one has type %v", *resultType, t)
		}

		switch t.Kind() {
		case reflect.Map:
			if mergedResults == nil {
				mergedResults = make(map[string]interface{})
			}
			mr := mergedResults.(map[string]interface{})
			for k, v := range res.(map[string]interface{}) {
				if v1, exist := mr[k]; exist {
					return nil, fmt.Errorf("merge conflict: multiple results define key %v: oldValue='%v', newValue='%v'", k, v1, v)
				}
				mr[k] = v
			}
		case reflect.Slice:
			if mergedResults == nil {
				mergedResults = make([]interface{}, 0)
			}
			for _, v := range res.([]interface{}) {
				mergedResults = append(mergedResults.([]interface{}), v)
			}
		default:
			return nil, fmt.Errorf("type %v is not supported for auto-merge, please provide a custom merge function", t)
		}
	}
	return
}
