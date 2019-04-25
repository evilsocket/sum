package main

import (
	"context"
	"fmt"
	. "github.com/evilsocket/sum/proto"
)

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
	ms.nextRaccoonId++

	ms.raccoons[raccoon.ID] = raccoon

	return &OracleResponse{Success: true, Msg: fmt.Sprintf("%d", raccoon.ID)}, nil
}

func (ms *MuxService) UpdateOracle(ctx context.Context, arg *Oracle) (*OracleResponse, error) {
	raccoon, err := NewAstRaccoon(arg.Code)
	if err != nil {
		return errOracleResponse("Error parsing the code: %v", err), nil
	}

	raccoon.Name = arg.Name

	ms.cageLock.Lock()
	defer ms.cageLock.Unlock()

	if _, found := ms.raccoons[arg.Id]; !found {
		return errOracleResponse("Oracle %d not found", arg.Id), nil
	}

	raccoon.ID = arg.Id
	ms.raccoons[arg.Id] = raccoon

	return &OracleResponse{Success: true}, nil
}

func (ms *MuxService) ReadOracle(ctx context.Context, arg *ById) (*OracleResponse, error) {
	ms.cageLock.RLock()
	defer ms.cageLock.RUnlock()

	if raccoon, found := ms.raccoons[arg.Id]; !found {
		return errOracleResponse("Oracle %d not found", arg.Id), nil
	} else {
		return &OracleResponse{Success: true, Oracles: []*Oracle{raccoon.AsOracle()}}, nil
	}
}

func (ms *MuxService) FindOracle(ctx context.Context, arg *ByName) (*OracleResponse, error) {
	ms.cageLock.RLock()
	defer ms.cageLock.RUnlock()

	var res = make([]*Oracle, 0)

	for _, r := range ms.raccoons {
		if r.Name == arg.Name {
			res = append(res, r.AsOracle())
		}
	}

	return &OracleResponse{Success: true, Oracles: res}, nil
}

func (ms *MuxService) DeleteOracle(ctx context.Context, arg *ById) (*OracleResponse, error) {
	ms.cageLock.Lock()
	defer ms.cageLock.Unlock()

	if _, found := ms.raccoons[arg.Id]; !found {
		return errOracleResponse("Oracle %d not found", arg.Id), nil
	}
	delete(ms.raccoons, arg.Id)
	return &OracleResponse{Success: true}, nil
}
