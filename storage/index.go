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

	// ErrInvalidId is returned when the system detects a collision of
	// identifiers, usually due to multiple Sum instances running on the
	// same data path.
	ErrInvalidId = errors.New("identifier is not unique")
	// ErrRecordNotFound is the 404 of Sum, it is returned when the storage
	// manager can't find an object mapped to the queried identifier.
	ErrRecordNotFound = errors.New("record not found")

	pathSep = string(os.PathSeparator)
)

// Index is a generic data structure used to map any types of protobuf
// encoded messages to unique integer identifiers and persist them on
// disk transparently.
type Index struct {
	sync.RWMutex
	dataPath string
	index    map[uint64]proto.Message
	nextId   uint64
	driver   Driver
}

// NOTE: pathSep is added if needed when the index object is created,
// this spares us a third string concatenation or worse a Sprintf call.
func (i *Index) pathForId(id uint64) string {
	return i.dataPath + strconv.FormatUint(id, 10) + DatFileExt
}

func (i *Index) pathFor(record proto.Message) string {
	return i.pathForId(i.driver.GetId(record))
}

// WithDriver creates a new Index object with the specified storage.Driver
// used to handle the specifics of the protobuf messages being handled
// but this instance of the index.
func WithDriver(dataPath string, driver Driver) *Index {
	if !strings.HasSuffix(dataPath, pathSep) {
		dataPath += pathSep
	}
	return &Index{
		dataPath: dataPath,
		index:    make(map[uint64]proto.Message),
		nextId:   1,
		driver:   driver,
	}
}

// Load enumerates files in the data folder while deserializing them
// and mapping them into the index by their identifiers.
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
			record := i.driver.Make()
			if err := Load(fileName, record); err != nil {
				return err
			}
			recId := i.driver.GetId(record)
			i.index[recId] = record
			if recId > i.nextId {
				i.nextId = recId + 1
			}
		}
	}

	return nil
}

// ForEach executes a callback passing as argument every
// element of the index.
func (i *Index) ForEach(cb func(record proto.Message)) {
	i.RLock()
	defer i.RUnlock()
	for _, record := range i.index {
		cb(record)
	}
}

// Size returns the number of elements stored in this index.
func (i *Index) Size() uint64 {
	i.RLock()
	defer i.RUnlock()
	return uint64(len(i.index))
}

// NextId sets the value for the integer identifier to use
// every future record. NOTE: This method is just for internal
// use and the only reason why it's exposed is because of unit
// tests, do not use.
func (i *Index) NextId(next uint64) {
	i.Lock()
	defer i.Unlock()
	i.nextId = next
}

// Create stores the profobuf message in the index, setting its
// identifier to a new, unique value. Once created the object
// will be used in memory and persisted on disk.
func (i *Index) Create(record proto.Message) error {
	i.Lock()
	defer i.Unlock()

	// make sure the id is unique and that we
	// are able to create the data file
	recId := i.nextId
	i.driver.SetId(record, recId)
	if _, found := i.index[recId]; found {
		return ErrInvalidId
	} else if err := Flush(record, i.pathForId(recId)); err != nil {
		return err
	}

	i.nextId++
	i.index[recId] = record

	return nil
}

// Update changes the contents of a stored object given a protobuf
// message with its identifier and the new values to use. This operation
// will flush the record on disk.
func (i *Index) Update(record proto.Message) error {
	i.Lock()
	defer i.Unlock()

	recId := i.driver.GetId(record)
	stored, found := i.index[recId]
	if !found {
		return ErrRecordNotFound
	} else if err := i.driver.Copy(stored, record); err != nil {
		return err
	}
	return Flush(stored, i.pathForId(recId))
}

// Find returns the instance of a stored object given its identifier,
// or nil if the object can not be found.
func (i *Index) Find(id uint64) proto.Message {
	i.RLock()
	defer i.RUnlock()

	record, found := i.index[id]
	if found {
		return record
	}
	return nil
}

// Delete removes a stored object from the index given its identifier,
// it will return the removed object itself if found, or nil.
// This operation will also remove the object data file from disk.
func (i *Index) Delete(id uint64) proto.Message {
	i.Lock()
	defer i.Unlock()

	record, found := i.index[id]
	if !found {
		return nil
	}

	delete(i.index, id)

	os.Remove(i.pathForId(id))

	return record
}
