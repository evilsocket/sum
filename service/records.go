package service

import (
	"fmt"

	pb "github.com/evilsocket/sum/proto"
	"golang.org/x/net/context"
)

func errRecordResponse(format string, args ...interface{}) *pb.RecordResponse {
	return &pb.RecordResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
}

func (s *Service) NumRecords() uint64 {
	return s.records.Size()
}

func (s *Service) CreateRecord(ctx context.Context, record *pb.Record) (*pb.RecordResponse, error) {
	if err := s.records.Create(record); err != nil {
		return errRecordResponse("%s", err), nil
	}
	return &pb.RecordResponse{Success: true, Msg: record.Id}, nil
}

func (s *Service) UpdateRecord(ctx context.Context, record *pb.Record) (*pb.RecordResponse, error) {
	if err := s.records.Update(record); err != nil {
		return errRecordResponse("%s", err), nil
	}
	return &pb.RecordResponse{Success: true}, nil
}

func (s *Service) ReadRecord(ctx context.Context, query *pb.ById) (*pb.RecordResponse, error) {
	record := s.records.Find(query.Id)
	if record == nil {
		return errRecordResponse("Record %s not found.", query.Id), nil
	}
	return &pb.RecordResponse{Success: true, Record: record}, nil
}

func (s *Service) DeleteRecord(ctx context.Context, query *pb.ById) (*pb.RecordResponse, error) {
	record := s.records.Delete(query.Id)
	if record == nil {
		return errRecordResponse("Record %s not found.", query.Id), nil
	}
	return &pb.RecordResponse{Success: true}, nil
}
