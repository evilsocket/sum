package storage

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	pb "github.com/evilsocket/sum/proto"
)

type Records struct {
	sync.RWMutex
	dataPath string
	records  map[string]*pb.Record
}

func LoadRecords(dataPath string) (*Records, error) {
	dataPath, files, err := ListPath(dataPath)
	if err != nil {
		return nil, err
	}

	records := make(map[string]*pb.Record)
	nfiles := len(files)

	if nfiles > 0 {
		log.Printf("Loading %d data files from %s ...", len(files), dataPath)
		for fileUUID, fileName := range files {
			record := new(pb.Record)
			if err := Load(fileName, record); err != nil {
				return nil, err
			}

			if record.Id != fileUUID {
				return nil, fmt.Errorf("File UUID is %s but record id is %r.", fileUUID, record.Id)
			}
			records[fileUUID] = record
		}
	}

	return &Records{
		dataPath: dataPath,
		records:  records,
	}, nil
}

func (r *Records) Size() uint64 {
	r.RLock()
	defer r.RUnlock()
	return uint64(len(r.records))
}

func (r *Records) pathFor(record *pb.Record) string {
	return filepath.Join(r.dataPath, record.Id) + DatFileExt
}

func (r *Records) Create(record *pb.Record) error {
	record.Id = NewID()

	r.Lock()
	defer r.Unlock()

	// make sure the id is unique
	if _, found := r.records[record.Id]; found == true {
		return fmt.Errorf("Record identifier %s violates the unicity constraint.", record.Id)
	}

	r.records[record.Id] = record

	return Flush(record, r.pathFor(record))
}

func (r *Records) Update(record *pb.Record) error {
	r.Lock()
	defer r.Unlock()

	stored, found := r.records[record.Id]
	if found == false {
		return fmt.Errorf("Record %s not found.", record.Id)
	}

	if record.Meta != nil {
		stored.Meta = record.Meta
	}

	if record.Data != nil {
		stored.Data = record.Data
	}

	return Flush(stored, r.pathFor(stored))
}

func (r *Records) Find(id string) *pb.Record {
	r.RLock()
	defer r.RUnlock()

	record, found := r.records[id]
	if found == true {
		return record
	}
	return nil
}

func (r *Records) Delete(id string) *pb.Record {
	r.Lock()
	defer r.Unlock()

	record, found := r.records[id]
	if found == false {
		return nil
	}

	delete(r.records, id)

	os.Remove(r.pathFor(record))

	return record
}
