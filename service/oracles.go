package service

import (
	"fmt"

	pb "github.com/evilsocket/sum/proto"
	"golang.org/x/net/context"
)

func errOracleResponse(format string, args ...interface{}) *pb.OracleResponse {
	return &pb.OracleResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
}

func (s *Service) NumOracles() uint64 {
	return s.oracles.Size()
}

func (s *Service) CreateOracle(ctx context.Context, oracle *pb.Oracle) (*pb.OracleResponse, error) {
	if err := s.oracles.Create(oracle); err != nil {
		return errOracleResponse("%s", err), nil
	}
	return &pb.OracleResponse{Success: true, Msg: fmt.Sprintf("%d", oracle.Id)}, nil
}

func (s *Service) UpdateOracle(ctx context.Context, oracle *pb.Oracle) (*pb.OracleResponse, error) {
	if err := s.oracles.Update(oracle); err != nil {
		return errOracleResponse("%s", err), nil
	}
	return &pb.OracleResponse{Success: true}, nil
}

func (s *Service) ReadOracle(ctx context.Context, query *pb.ById) (*pb.OracleResponse, error) {
	compiled := s.oracles.Find(query.Id)
	if compiled == nil {
		return errOracleResponse("Oracle %d not found.", query.Id), nil
	}
	return &pb.OracleResponse{Success: true, Oracles: []*pb.Oracle{compiled.Oracle()}}, nil
}

func (s *Service) FindOracle(ctx context.Context, query *pb.ByName) (*pb.OracleResponse, error) {
	oracles := make([]*pb.Oracle, 0)
	s.oracles.ForEach(func(oracle *pb.Oracle) {
		if oracle.Name == query.Name {
			oracles = append(oracles, oracle)
		}
	})
	return &pb.OracleResponse{Success: true, Oracles: oracles}, nil
}

func (s *Service) DeleteOracle(ctx context.Context, query *pb.ById) (*pb.OracleResponse, error) {
	oracle := s.oracles.Delete(query.Id)
	if oracle == nil {
		return errOracleResponse("Oracle %d not found.", query.Id), nil
	}
	return &pb.OracleResponse{Success: true}, nil
}
