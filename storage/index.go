package storage

import (
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
)

var (
	ErrInvalidId      = errors.New("identifier is not unique")
	ErrRecordNotFound = errors.New("record not found")

	pathSep = string(os.PathSeparator)
)

type Maker func() proto.Message
type Hasher func(record proto.Message) uint64
type Marker func(record proto.Message, mark uint64)
type Copier func(old proto.Message, new proto.Message) error

// Index is a generic thread safe data structure used to
// map objects to unique integer identifiers.
type Index struct {
	sync.RWMutex
	dataPath string
	index    map[uint64]proto.Message
	nextId   uint64
	maker    Maker
	hasher   Hasher
	marker   Marker
	copier   Copier
}

func NewIndex(dataPath string) *Index {
	if strings.HasSuffix(dataPath, pathSep) == false {
		dataPath += pathSep
	}
	return &Index{
		dataPath: dataPath,
		index:    make(map[uint64]proto.Message),
		nextId:   1,
		maker:    nil,
		hasher:   nil,
		marker:   nil,
		copier:   nil,
	}
}

func (i *Index) Maker(maker Maker) {
	i.Lock()
	defer i.Unlock()
	i.maker = maker
}

func (i *Index) Hasher(hasher Hasher) {
	i.Lock()
	defer i.Unlock()
	i.hasher = hasher
}

func (i *Index) Marker(marker Marker) {
	i.Lock()
	defer i.Unlock()
	i.marker = marker
}

func (i *Index) Copier(copier Copier) {
	i.Lock()
	defer i.Unlock()
	i.copier = copier
}

func (i *Index) Load() error {
	i.Lock()
	defer i.Unlock()

	absPath, files, err := ListPath(i.dataPath)
	if err != nil {
		return err
	}

	i.dataPath = absPath + pathSep
	i.nextId = 1
	if nfiles := len(files); nfiles > 0 {
		log.Printf("loading %d data files from %s ...", len(files), i.dataPath)
		for _, fileName := range files {
			record := i.maker()
			if err := Load(fileName, record); err != nil {
				return err
			}
			recId := i.hasher(record)
			i.index[recId] = record
			if recId > i.nextId {
				i.nextId = recId + 1
			}
		}
	}

	return nil
}

func (i *Index) pathForId(id uint64) string {
	return i.dataPath + strconv.FormatUint(id, 10) + DatFileExt
}

func (i *Index) pathFor(record proto.Message) string {
	return i.pathForId(i.hasher(record))
}

func (i *Index) ForEach(cb func(record proto.Message)) {
	i.RLock()
	defer i.RUnlock()
	for _, record := range i.index {
		cb(record)
	}
}

func (i *Index) Size() uint64 {
	i.RLock()
	defer i.RUnlock()
	return uint64(len(i.index))
}

func (i *Index) NextId(next uint64) {
	i.Lock()
	defer i.Unlock()
	i.nextId = next
}

func (i *Index) Create(record proto.Message) error {
	i.Lock()
	defer i.Unlock()

	// make sure the id is unique and that we
	// are able to create the data file
	recId := i.nextId
	i.marker(record, recId)
	if _, found := i.index[recId]; found == true {
		return ErrInvalidId
	} else if err := Flush(record, i.pathForId(recId)); err != nil {
		return err
	}

	i.nextId++
	i.index[recId] = record

	return nil
}

func (i *Index) Update(record proto.Message) error {
	i.Lock()
	defer i.Unlock()

	recId := i.hasher(record)
	stored, found := i.index[recId]
	if found == false {
		return ErrRecordNotFound
	} else if err := i.copier(stored, record); err != nil {
		return err
	}
	return Flush(stored, i.pathForId(recId))
}

func (i *Index) Find(id uint64) proto.Message {
	i.RLock()
	defer i.RUnlock()

	record, found := i.index[id]
	if found == true {
		return record
	}
	return nil
}

func (i *Index) Delete(id uint64) proto.Message {
	i.Lock()
	defer i.Unlock()

	record, found := i.index[id]
	if found == false {
		return nil
	}

	delete(i.index, id)

	os.Remove(i.pathForId(id))

	return record
}
