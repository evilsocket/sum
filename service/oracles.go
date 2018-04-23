package service

import (
	"fmt"

	pb "github.com/evilsocket/sum/proto"
	"github.com/golang/protobuf/proto"

	"golang.org/x/net/context"
)

func errOracleResponse(format string, args ...interface{}) *pb.OracleResponse {
	return &pb.OracleResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
}

// NumOracles returns the number of oracles currently loaded by the service.
func (s *Service) NumOracles() int {
	return s.oracles.Size()
}

// CreateOracle compiles and stores a raw *pb.Oracle object. If successful, the
// identifier of the newly created oracle is returned as the response message.
func (s *Service) CreateOracle(ctx context.Context, oracle *pb.Oracle) (*pb.OracleResponse, error) {
	if compiled, err := compile(oracle); err != nil {
		return errOracleResponse("%s", err), nil
	} else if err := s.oracles.Create(oracle); err != nil {
		return errOracleResponse("%s", err), nil
	} else {
		s.cache.Add(oracle.Id, compiled)
	}
	return &pb.OracleResponse{Success: true, Msg: fmt.Sprintf("%d", oracle.Id)}, nil
}

// UpdateOracle updates the contents of an oracle with the ones of a raw *pb.Oracle
// object given its identifier.
func (s *Service) UpdateOracle(ctx context.Context, oracle *pb.Oracle) (*pb.OracleResponse, error) {
	if compiled, err := compile(oracle); err != nil {
		return errOracleResponse("%s", err), nil
	} else if err := s.oracles.Update(oracle); err != nil {
		return errOracleResponse("%s", err), nil
	} else {
		s.cache.Add(oracle.Id, compiled)
	}
	return &pb.OracleResponse{Success: true}, nil
}

// ReadOracle returns a raw *pb.Oracle object given its identifier.
func (s *Service) ReadOracle(ctx context.Context, query *pb.ById) (*pb.OracleResponse, error) {
	oracle := s.oracles.Find(query.Id)
	if oracle == nil {
		return errOracleResponse("oracle %d not found.", query.Id), nil
	}
	return &pb.OracleResponse{Success: true, Oracles: []*pb.Oracle{oracle}}, nil
}

// FindOracle returns a list of raw *pb.Oracle objects that match
// the provided name.
func (s *Service) FindOracle(ctx context.Context, query *pb.ByName) (*pb.OracleResponse, error) {
	oracles := make([]*pb.Oracle, 0)
	s.oracles.ForEach(func(m proto.Message) error {
		if oracle := m.(*pb.Oracle); oracle.Name == query.Name {
			oracles = append(oracles, oracle)
		}
		return nil
	})
	return &pb.OracleResponse{Success: true, Oracles: oracles}, nil
}

// DeleteOracle removes an oracle from the storage given its identifier.
func (s *Service) DeleteOracle(ctx context.Context, query *pb.ById) (*pb.OracleResponse, error) {
	if oracle := s.oracles.Delete(query.Id); oracle == nil {
		return errOracleResponse("Oracle %d not found.", query.Id), nil
	}
	s.cache.Del(query.Id)
	return &pb.OracleResponse{Success: true}, nil
}
