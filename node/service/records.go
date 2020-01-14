package service

import (
	"fmt"
	"sort"

	pb "github.com/evilsocket/sum/proto"
	"golang.org/x/net/context"
)

func errRecordResponse(format string, args ...interface{}) *pb.RecordResponse {
	return &pb.RecordResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
}

func errFindResponse(format string, args ...interface{}) *pb.FindResponse {
	return &pb.FindResponse{Success: false, Msg: fmt.Sprintf(format, args...)}
}

// NumRecords returns the number of records currently loaded by the service.
func (s *Service) NumRecords() int {
	return s.records.Size()
}

// CreateRecord creates and stores a new *pb.Record object. If successful, the
// identifier of the record is returned as the response message.
func (s *Service) CreateRecord(ctx context.Context, record *pb.Record) (*pb.RecordResponse, error) {
	if err := s.records.Create(record); err != nil {
		return errRecordResponse("%s", err), nil
	}
	return &pb.RecordResponse{Success: true, Msg: fmt.Sprintf("%d", record.Id)}, nil
}

// CreateRecords creates and stores a series of new *pb.Record object.
func (s *Service) CreateRecords(ctx context.Context, records *pb.Records) (*pb.RecordResponse, error) {
	if err := s.records.CreateMulti(records); err != nil {
		return errRecordResponse("%s", err), nil
	}
	return &pb.RecordResponse{Success: true}, nil
}

func (s *Service) CreateRecordWithId(ctx context.Context, record *pb.Record) (*pb.RecordResponse, error) {
	if err := s.records.CreateWithId(record); err != nil {
		return errRecordResponse("%s", err), nil
	}
	return &pb.RecordResponse{Success: true, Msg: fmt.Sprintf("%d", record.Id)}, nil
}

func (s *Service) CreateRecordsWithId(ctx context.Context, records *pb.Records) (*pb.RecordResponse, error) {
	if err := s.records.CreateManyWIthId(records.Records); err != nil {
		return errRecordResponse("%v", err), nil
	}
	return &pb.RecordResponse{Success: true}, nil
}

// UpdateRecord updates the contents of a record with the ones of a raw *pb.Record
// object given its identifier.
func (s *Service) UpdateRecord(ctx context.Context, record *pb.Record) (*pb.RecordResponse, error) {
	if err := s.records.Update(record); err != nil {
		return errRecordResponse("%s", err), nil
	}
	return &pb.RecordResponse{Success: true}, nil
}

// ReadRecord returns a raw *pb.Record object given its identifier.
func (s *Service) ReadRecord(ctx context.Context, query *pb.ById) (*pb.RecordResponse, error) {
	record := s.records.Find(query.Id)
	if record == nil {
		return errRecordResponse("record %d not found.", query.Id), nil
	}
	return &pb.RecordResponse{Success: true, Record: record}, nil
}

// ListRecords returns list of records given a ListRequest object.
func (s *Service) ListRecords(ctx context.Context, list *pb.ListRequest) (*pb.RecordListResponse, error) {
	all := s.records.Objects()
	total := uint64(len(all))

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
		return &pb.RecordListResponse{Total: total, Pages: npages}, nil
	}

	resp := pb.RecordListResponse{
		Total:   total,
		Pages:   npages,
		Records: make([]*pb.Record, 0),
	}

	// sort by id
	sort.Slice(all, func(i, j int) bool {
		return all[i].(*pb.Record).Id < all[j].(*pb.Record).Id
	})

	if total <= end {
		// partially filled page
		for _, m := range all[start:] {
			resp.Records = append(resp.Records, m.(*pb.Record))
		}
	} else {
		// full page
		for _, m := range all[start:end] {
			resp.Records = append(resp.Records, m.(*pb.Record))
		}
	}

	return &resp, nil
}

// DeleteRecord removes a record from the storage given its identifier.
func (s *Service) DeleteRecord(ctx context.Context, query *pb.ById) (*pb.RecordResponse, error) {
	record := s.records.Delete(query.Id)
	if record == nil {
		return errRecordResponse("record %d not found.", query.Id), nil
	}
	return &pb.RecordResponse{Success: true}, nil
}

func (s *Service) DeleteRecords(ctx context.Context, ids *pb.RecordIds) (*pb.RecordResponse, error) {
	s.records.DeleteMany(ids.Ids)
	return &pb.RecordResponse{Success: true}, nil
}

// FindRecords returns a FindResponse object corresponding to the records that matched the search criteria.
func (s *Service) FindRecords(ctx context.Context, query *pb.ByMeta) (*pb.FindResponse, error) {
	records := s.records.FindBy(query.Meta, query.Value)
	if records == nil {
		return errFindResponse("meta %s not indexed.", query.Meta), nil
	}
	return &pb.FindResponse{Success: true, Records: records}, nil
}
