package wrapper

import (
	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/storage"

	"github.com/golang/protobuf/proto"
)

// Records is an object that wraps *storage.Records in
// order to give access to those records to oracles during
// execution.
type Records struct {
	records *storage.Records
}

// WrapRecords creates a Records wrapper around a *storage.Records object.
func WrapRecords(records *storage.Records) Records {
	return Records{
		records: records,
	}
}

// Find returns a wrapped Record given its identifier.
// If not found, the resulting record will result as null
// (record.IsNull() will be true).
func (w Records) Find(id uint64) Record {
	return WrapRecord(w.records.Find(id))
}

// All returns a wrapped list of records in the current storage.
func (w Records) All() []Record {
	wrapped := make([]Record, w.records.Size())
	idx := 0
	w.records.ForEach(func(m proto.Message) error {
		wrapped[idx] = WrapRecord(m.(*pb.Record))
		idx++
		return nil
	})
	return wrapped
}

// AllBut returns a wrapped list of records in the current storage
// but the one specified.
func (w Records) AllBut(exclude Record) []Record {
	// NOTE: this preallocation assumes the excluded element will
	// be found in the list of records.
	wrapped := make([]Record, w.records.Size()-1)
	idx := 0
	w.records.ForEach(func(m proto.Message) error {
		record := m.(*pb.Record)
		if record.Id != exclude.record.Id {
			wrapped[idx] = WrapRecord(record)
			idx++
		}
		return nil
	})
	return wrapped
}
