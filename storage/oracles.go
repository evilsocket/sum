package storage

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	pb "github.com/evilsocket/sum/proto"
)

type Oracles struct {
	sync.RWMutex
	dataPath string
	index    map[string]*pb.Oracle
}

func LoadOracles(dataPath string) (*Oracles, error) {
	dataPath, files, err := ListPath(dataPath)
	if err != nil {
		return nil, err
	}

	oracles := make(map[string]*pb.Oracle)
	nfiles := len(files)

	if nfiles > 0 {
		log.Printf("Loading %d data files from %s ...", len(files), dataPath)
		for fileUUID, fileName := range files {
			oracle := new(pb.Oracle)
			if err := Load(fileName, oracle); err != nil {
				return nil, err
			}

			if oracle.Id != fileUUID {
				return nil, fmt.Errorf("File UUID is %s but oracle id is %r.", fileUUID, oracle.Id)
			}

			// TODO: compile

			oracles[fileUUID] = oracle
		}
	}

	return &Oracles{
		dataPath: dataPath,
		index:    oracles,
	}, nil
}

func (o *Oracles) Size() uint64 {
	o.RLock()
	defer o.RUnlock()
	return uint64(len(o.index))
}

func (o *Oracles) pathFor(oracle *pb.Oracle) string {
	return filepath.Join(o.dataPath, oracle.Id) + DatFileExt
}

func (o *Oracles) Create(oracle *pb.Oracle) error {
	oracle.Id = NewID()

	o.Lock()
	defer o.Unlock()

	// make sure the id is unique
	if _, found := o.index[oracle.Id]; found == true {
		return fmt.Errorf("Oracle identifier %s violates the unicity constraint.", oracle.Id)
	}

	o.index[oracle.Id] = oracle

	return Flush(oracle, o.pathFor(oracle))
}

func (o *Oracles) Update(oracle *pb.Oracle) error {
	o.Lock()
	defer o.Unlock()

	stored, found := o.index[oracle.Id]
	if found == false {
		return fmt.Errorf("Oracle %s not found.", oracle.Id)
	}

	stored.Name = oracle.Name
	stored.Code = oracle.Code

	// TODO: compile

	return Flush(stored, o.pathFor(stored))
}

func (o *Oracles) Find(id string) *pb.Oracle {
	o.RLock()
	defer o.RUnlock()

	oracle, found := o.index[id]
	if found == true {
		return oracle
	}
	return nil
}

func (o *Oracles) Delete(id string) *pb.Oracle {
	o.Lock()
	defer o.Unlock()

	oracle, found := o.index[id]
	if found == false {
		return nil
	}

	delete(o.index, id)

	os.Remove(o.pathFor(oracle))

	return oracle
}
