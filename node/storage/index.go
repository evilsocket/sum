package storage

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"

	"github.com/evilsocket/islazy/log"
)

var (

	// ErrInvalidID is returned when the system detects a collision of
	// identifiers, usually due to multiple Sum instances running on the
	// same data path.
	ErrInvalidID = errors.New("identifier is not unique")
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
	nextID   uint64
	driver   Driver
}

func (i *Index) GetNextId() uint64 {
	i.RLock()
	defer i.RUnlock()
	return i.nextID
}

// NOTE: pathSep is added if needed when the index object is created,
// this spares us a third string concatenation or worse a Sprintf call.
func (i *Index) pathForID(id uint64) string {
	return i.dataPath + strconv.FormatUint(id, 10) + DatFileExt
}

func (i *Index) pathFor(record proto.Message) string {
	return i.pathForID(i.driver.GetID(record))
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
		nextID:   1,
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
	i.nextID = 1
	if nfiles := len(files); nfiles > 0 {
		log.Info("loading %d data files from %s ...", len(files), i.dataPath)
		for _, fileName := range files {
			record := i.driver.Make()
			if err := Load(fileName, record); err != nil {
				return err
			}
			recID := i.driver.GetID(record)
			i.index[recID] = record
			// the list 'files' returned by ListPath is not sorted,
			// so if the last 2 loaded files have the last sequential ids (4 and 5 for example)
			// the id will be increased with the second-last record but not with the last one.
			if recID >= i.nextID {
				i.nextID = recID + 1
			}
		}
	}

	return nil
}

// ForEach executes a callback passing as argument every
// element of the index, it interrupts the loop if the
// callback returns an error, the same error will be returned.
func (i *Index) ForEach(cb func(record proto.Message) error) error {
	i.RLock()
	defer i.RUnlock()
	for _, record := range i.index {
		if err := cb(record); err != nil {
			return err
		}
	}
	return nil
}

// Objects return a list of proto.Message objects stored in this
// index.
func (i *Index) Objects() []proto.Message {
	i.RLock()
	defer i.RUnlock()

	numObjects := len(i.index)
	asSlice := make([]proto.Message, numObjects)
	idx := 0
	for _, record := range i.index {
		asSlice[idx] = record
		idx++
	}
	return asSlice
}

// Size returns the number of elements stored in this index.
func (i *Index) Size() int {
	i.RLock()
	defer i.RUnlock()
	return len(i.index)
}

// NextID sets the value for the integer identifier to use
// every future record. NOTE: This method is just for internal
// use and the only reason why it's exposed is because of unit
// tests, do not use.
func (i *Index) NextID(next uint64) {
	i.Lock()
	defer i.Unlock()
	i.nextID = next
}

// Create stores the profobuf message in the index, setting its
// identifier to a new, unique value. Once created the object
// will be used in memory and persisted on disk.
func (i *Index) Create(record proto.Message) error {
	i.Lock()
	defer i.Unlock()

	// make sure the id is unique and that we
	// are able to create the data file
	recID := i.nextID
	i.driver.SetID(record, recID)
	if _, found := i.index[recID]; found {
		return ErrInvalidID
	} else if err := Flush(record, i.pathForID(recID)); err != nil {
		return err
	}

	i.nextID++
	i.index[recID] = record

	return nil
}

func (i *Index) CreateWithId(record proto.Message) error {
	i.Lock()
	defer i.Unlock()

	recID := i.driver.GetID(record)
	if _, found := i.index[recID]; found {
		return ErrInvalidID
	} else if err := Flush(record, i.pathForID(recID)); err != nil {
		return err
	}

	i.index[recID] = record

	return nil
}

func (i *Index) CreateManyWIthId(records []proto.Message) (err error) {
	rollbackOnError := func(e *error) {
		if *e == nil {
			return
		}
		// rollback
		for _, record := range records {
			id := i.driver.GetID(record)
			delete(i.index, id)
			os.Remove(i.pathForID(id))
		}
	}

	i.Lock()
	defer i.Unlock()

	defer rollbackOnError(&err)

	for _, record := range records {
		id := i.driver.GetID(record)
		if err = Flush(record, i.pathForID(id)); err != nil {
			break
		}

		i.index[id] = record
	}

	return
}

// Update changes the contents of a stored object given a protobuf
// message with its identifier and the new values to use. This operation
// will flush the record on disk.
func (i *Index) Update(record proto.Message) error {
	i.Lock()
	defer i.Unlock()

	recID := i.driver.GetID(record)
	stored, found := i.index[recID]
	if !found {
		return ErrRecordNotFound
	} else if err := i.driver.Copy(stored, record); err != nil {
		return err
	}
	return Flush(stored, i.pathForID(recID))
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

	os.Remove(i.pathForID(id))

	return record
}

// Delete multiple records at once
func (i *Index) DeleteMany(ids []uint64) []proto.Message {
	res := make([]proto.Message, 0, len(ids))

	i.Lock()
	defer i.Unlock()

	for _, id := range ids {
		record, found := i.index[id]
		if !found {
			continue
		}
		delete(i.index, id)
		os.Remove(i.pathForID(id))
		res = append(res, record)
	}

	return res
}
