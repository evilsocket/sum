package main

import (
	"context"
	"fmt"
	. "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/storage"
	"sort"
)

// create an oracle form the given argument
func (ms *MuxService) CreateOracle(ctx context.Context, arg *Oracle) (*OracleResponse, error) {
	raccoon, err := NewAstRaccoon(arg.Code)
	if err != nil {
		return errOracleResponse("Error parsing the code: %v", err), nil
	}

	raccoon.Name = arg.Name

	// store the raccoon

	ms.cageLock.Lock()
	defer ms.cageLock.Unlock()

	raccoon.ID = ms.nextRaccoonId

	if _, exists := ms.raccoons[raccoon.ID]; exists {
		return errOracleResponse("%v", storage.ErrInvalidID), nil
	}

	ms.raccoons[raccoon.ID] = raccoon

	ms.nextRaccoonId++

	return &OracleResponse{Success: true, Msg: fmt.Sprintf("%d", raccoon.ID)}, nil
}

// update an oracle from the given argument
func (ms *MuxService) UpdateOracle(ctx context.Context, arg *Oracle) (*OracleResponse, error) {
	raccoon, err := NewAstRaccoon(arg.Code)
	if err != nil {
		return errOracleResponse("Error parsing the code: %v", err), nil
	}

	raccoon.Name = arg.Name

	ms.cageLock.Lock()
	defer ms.cageLock.Unlock()

	if _, found := ms.raccoons[arg.Id]; !found {
		return errOracleResponse("%v", storage.ErrRecordNotFound), nil
	}

	raccoon.ID = arg.Id
	ms.raccoons[arg.Id] = raccoon

	return &OracleResponse{Success: true}, nil
}

// retrieve an Oracle's content
func (ms *MuxService) ReadOracle(ctx context.Context, arg *ById) (*OracleResponse, error) {
	ms.cageLock.RLock()
	defer ms.cageLock.RUnlock()

	if raccoon, found := ms.raccoons[arg.Id]; !found {
		return errOracleResponse("oracle %d not found.", arg.Id), nil
	} else {
		return &OracleResponse{Success: true, Oracle: raccoon.AsOracle()}, nil
	}
}

// Find an Oracle by it's name
func (ms *MuxService) FindOracle(ctx context.Context, arg *ByName) (*OracleResponse, error) {
	ms.cageLock.RLock()
	defer ms.cageLock.RUnlock()

	for _, r := range ms.raccoons {
		if r.Name == arg.Name {
			return &OracleResponse{Success: true, Oracle: r.AsOracle()}, nil
		}
	}

	return errOracleResponse("oracle '%s' not found.", arg.Name), nil
}

// List oracles
func (ms *MuxService) ListOracles(ctx context.Context, list *ListRequest) (*OracleListResponse, error) {
	ms.cageLock.RLock()
	defer ms.cageLock.RUnlock()

	sortedIds := make([]uint64, 0, len(ms.raccoons))

	for id := range ms.raccoons {
		sortedIds = append(sortedIds, id)
	}

	sort.Slice(sortedIds, func(i, j int) bool { return i < j })

	total := uint64(len(sortedIds))

	if list.Page < 1 {
		list.Page = 1
	}

	if list.PerPage < 1 {
		list.PerPage = 1
	}

	start := (list.Page - 1) * list.PerPage
	end := start + list.PerPage
	npages := total / list.PerPage
	if total%list.PerPage > 0 {
		npages++
	}

	// out of range
	if total <= start {
		return &OracleListResponse{Total: total, Pages: npages}, nil
	}

	resp := OracleListResponse{
		Total:   total,
		Pages:   npages,
		Oracles: make([]*Oracle, 0),
	}

	if end >= total {
		end = total - 1
	}

	for _, id := range sortedIds[start:end] {
		r := ms.raccoons[id]
		resp.Oracles = append(resp.Oracles, r.AsOracle())
	}

	return &resp, nil
}

// delete the specified oracle
func (ms *MuxService) DeleteOracle(ctx context.Context, arg *ById) (*OracleResponse, error) {
	ms.cageLock.Lock()
	defer ms.cageLock.Unlock()

	if _, found := ms.raccoons[arg.Id]; !found {
		return errOracleResponse("Oracle %d not found.", arg.Id), nil
	}
	delete(ms.raccoons, arg.Id)
	return &OracleResponse{Success: true}, nil
}
