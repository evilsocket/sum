package storage

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	pb "github.com/evilsocket/sum/proto"

	"github.com/robertkrimen/otto"
)

type Oracles struct {
	sync.RWMutex
	dataPath string
	vm       *otto.Otto
	index    map[string]*CompiledOracle
}

func LoadOracles(dataPath string) (*Oracles, error) {
	dataPath, files, err := ListPath(dataPath)
	if err != nil {
		return nil, err
	}

	oracles := make(map[string]*CompiledOracle)
	nfiles := len(files)
	vm := otto.New()

	if nfiles > 0 {
		log.Printf("Loading %d data files from %s ...", len(files), dataPath)
		for fileUUID, fileName := range files {
			oracle := new(pb.Oracle)
			if err := Load(fileName, oracle); err != nil {
				return nil, err
			} else if oracle.Id != fileUUID {
				return nil, fmt.Errorf("File UUID is %s but oracle id is %s.", fileUUID, oracle.Id)
			} else if compiled, err := Compile(vm, oracle); err != nil {
				return nil, fmt.Errorf("Error compiling oracle %s: %s", fileUUID, err)
			} else {
				oracles[fileUUID] = compiled
			}
		}
	}

	return &Oracles{
		dataPath: dataPath,
		index:    oracles,
		vm:       vm,
	}, nil
}

func (o *Oracles) VM() *otto.Otto {
	return o.vm
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

	compiled, err := Compile(o.vm, oracle)
	if err != nil {
		return fmt.Errorf("Error compiling oracle %s: %s", oracle.Id, err)
	}

	o.index[oracle.Id] = compiled
	return Flush(oracle, o.pathFor(oracle))
}

func (o *Oracles) Update(oracle *pb.Oracle) (err error) {
	o.Lock()
	defer o.Unlock()

	compiled, found := o.index[oracle.Id]
	if found == false {
		return fmt.Errorf("Oracle %s not found.", oracle.Id)
	}

	if compiled, err = Compile(o.vm, oracle); err != nil {
		return fmt.Errorf("Error compiling oracle %s: %s", oracle.Id, err)
	}

	o.index[oracle.Id] = compiled

	return Flush(oracle, o.pathFor(oracle))
}

func (o *Oracles) Find(id string) *CompiledOracle {
	o.RLock()
	defer o.RUnlock()

	if compiled, found := o.index[id]; found == true {
		return compiled
	}
	return nil
}

func (o *Oracles) Delete(id string) *pb.Oracle {
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
