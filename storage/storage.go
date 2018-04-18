package storage

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	pb "github.com/evilsocket/sum/proto"
	"github.com/golang/protobuf/proto"

	"github.com/satori/go.uuid"
)

const (
	DatFileExt = ".dat"
)

type Storage struct {
	sync.RWMutex
	dataPath string
	records  map[string]*pb.Record
}

func NewID() string {
	return uuid.Must(uuid.NewV4()).String()
}

func New(dataPath string) (*Storage, error) {
	if dataPath, err := filepath.Abs(dataPath); err != nil {
		return nil, err
	} else if info, err := os.Stat(dataPath); err != nil {
		return nil, err
	} else if info.IsDir() == false {
		return nil, fmt.Errorf("%s is not a folder.", dataPath)
	}

	files, err := ioutil.ReadDir(dataPath)
	if err != nil {
		return nil, err
	}

	records := make(map[string]*pb.Record)
	for _, file := range files {
		fileName := file.Name()
		fileExt := filepath.Ext(fileName)
		if fileExt != DatFileExt {
			continue
		}

		fileUUID := strings.Replace(fileName, DatFileExt, "", -1)
		if _, err := uuid.FromString(fileUUID); err == nil {
			fileName = filepath.Join(dataPath, fileName)
			log.Printf("loading data file %s ...", fileName)

			data, err := ioutil.ReadFile(fileName)
			if err != nil {
				return nil, fmt.Errorf("Error while reading %s: %s", fileName, err)
			}

			record := new(pb.Record)
			err = proto.Unmarshal(data, record)
			if err != nil {
				return nil, fmt.Errorf("Error while deserializing %s: %s", fileName, err)
			}

			if record.Id != fileUUID {
				return nil, fmt.Errorf("File UUID is %s but record id is %s.", fileUUID, record.Id)
			}

			records[fileUUID] = record
		}
	}

	return &Storage{
		dataPath: dataPath,
		records:  records,
	}, nil
}

func (s *Storage) Size() uint64 {
	s.RLock()
	defer s.RUnlock()
	return uint64(len(s.records))
}

func (s *Storage) pathFor(record *pb.Record) string {
	return filepath.Join(s.dataPath, record.Id) + DatFileExt
}

func (s *Storage) flushUnlocked(record *pb.Record) error {
	data, err := proto.Marshal(record)
	if err != nil {
		return fmt.Errorf("Error while serializing record %s: %s", record.Id, err)
	} else if err = ioutil.WriteFile(s.pathFor(record), data, 0755); err != nil {
		return fmt.Errorf("Error while saving record %s: %s", record.Id, err)
	}
	return nil
}

func (s *Storage) Create(record *pb.Record) error {
	record.Id = NewID()

	s.Lock()
	defer s.Unlock()

	// make sure the id is unique
	if _, found := s.records[record.Id]; found == true {
		return fmt.Errorf("Identifier %s violates the unicity constraint.", record.Id)
	}

	s.records[record.Id] = record

	return s.flushUnlocked(record)
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

	return s.flushUnlocked(stored)
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

	os.Remove(s.pathFor(record))

	return record
}
