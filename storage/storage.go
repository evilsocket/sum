package storage

import (
	"fmt"
	"sync"

	pb "github.com/evilsocket/sum/proto"

	"github.com/satori/go.uuid"
)

type Storage struct {
	sync.RWMutex
	records map[string]*pb.Record
}

func NewID() string {
	return uuid.Must(uuid.NewV4()).String()
}

func New() *Storage {
	return &Storage{
		records: make(map[string]*pb.Record),
	}
}

func (s *Storage) Size() uint64 {
	s.RLock()
	defer s.RUnlock()
	return uint64(len(s.records))
}

func (s *Storage) Create(record *pb.Record) error {
	// if no id was filled, generate a new one
	if record.Id == "" {
		record.Id = NewID()
	}

	s.Lock()
	defer s.Unlock()

	// make sure the id is unique
	if _, found := s.records[record.Id]; found == true {
		return fmt.Errorf("Identifier %s violates the unicity constraint.", record.Id)
	}

	s.records[record.Id] = record

	return nil
}

func (s *Storage) Update(record *pb.Record) error {
	s.Lock()
	defer s.Unlock()

	stored, found := s.records[record.Id]
	if found == false {
		return fmt.Errorf("Record %s not found.", record.Id)
	}

	if record.Meta != nil {
		stored.Meta = record.Meta
	}

	if record.Data != nil {
		stored.Data = record.Data
	}

	return nil
}

func (s *Storage) Find(id string) *pb.Record {
	s.RLock()
	defer s.RUnlock()

	record, found := s.records[id]
	if found == true {
		return record
	}
	return nil
}

func (s *Storage) Delete(id string) *pb.Record {
	s.Lock()
	defer s.Unlock()

	record, found := s.records[id]
	if found == false {
		return nil
	}

	delete(s.records, id)

	return record
}
