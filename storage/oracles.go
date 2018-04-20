package storage

import (
	"fmt"
	"log"
	"os"
	"sync"

	pb "github.com/evilsocket/sum/proto"
)

type Oracles struct {
	sync.RWMutex
	dataPath string
	index    map[uint64]*CompiledOracle
	nextId   uint64
}

func LoadOracles(dataPath string) (*Oracles, error) {
	dataPath, files, err := ListPath(dataPath)
	if err != nil {
		return nil, err
	}

	oracles := make(map[uint64]*CompiledOracle)
	nfiles := len(files)
	maxId := uint64(0)

	if nfiles > 0 {
		log.Printf("Loading %d data files from %s ...", len(files), dataPath)
		for _, fileName := range files {
			oracle := new(pb.Oracle)
			if err := Load(fileName, oracle); err != nil {
				return nil, err
			} else if compiled, err := Compile(oracle); err != nil {
				return nil, fmt.Errorf("Error compiling oracle %d: %s", oracle.Id, err)
			} else {
				oracles[oracle.Id] = compiled
				if oracle.Id > maxId {
					maxId = oracle.Id
				}
			}
		}
	}

	return &Oracles{
		dataPath: dataPath,
		index:    oracles,
		nextId:   maxId + 1,
	}, nil
}

func (o *Oracles) ForEach(cb func(oracle *pb.Oracle)) {
	o.RLock()
	defer o.RUnlock()
	for _, compiled := range o.index {
		cb(compiled.oracle)
	}
}

func (o *Oracles) Size() uint64 {
	o.RLock()
	defer o.RUnlock()
	return uint64(len(o.index))
}

func (o *Oracles) pathFor(oracle *pb.Oracle) string {
	return o.dataPath + fmt.Sprintf("/%d.dat", oracle.Id)
}

func (o *Oracles) Create(oracle *pb.Oracle) error {
	o.Lock()
	defer o.Unlock()

	oracle.Id = o.nextId
	o.nextId++

	// make sure the id is unique
	if _, found := o.index[oracle.Id]; found == true {
		return fmt.Errorf("Oracle identifier %d violates the unicity constraint.", oracle.Id)
	}

	compiled, err := Compile(oracle)
	if err != nil {
		return fmt.Errorf("Error compiling oracle %d: %s", oracle.Id, err)
	}

	o.index[oracle.Id] = compiled
	return Flush(oracle, o.pathFor(oracle))
}

func (o *Oracles) Update(oracle *pb.Oracle) (err error) {
	o.Lock()
	defer o.Unlock()

	compiled, found := o.index[oracle.Id]
	if found == false {
		return fmt.Errorf("Oracle %d not found.", oracle.Id)
	}

	if compiled, err = Compile(oracle); err != nil {
		return fmt.Errorf("Error compiling oracle %d: %s", oracle.Id, err)
	}

	o.index[oracle.Id] = compiled

	return Flush(oracle, o.pathFor(oracle))
}

func (o *Oracles) Find(id uint64) *CompiledOracle {
	o.RLock()
	defer o.RUnlock()

	if compiled, found := o.index[id]; found == true {
		return compiled
	}
	return nil
}

func (o *Oracles) Delete(id uint64) *pb.Oracle {
	o.Lock()
	defer o.Unlock()

	compiled, found := o.index[id]
	if found == false {
		return nil
	}

	delete(o.index, id)

	os.Remove(o.pathFor(compiled.oracle))

	return compiled.oracle
}
