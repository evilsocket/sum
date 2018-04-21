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
	return WrapRecord(w.records, w.records.Find(id))
}

// All returns a wrapped list of records in the current storage.
func (w Records) All() []Record {
	wrapped := make([]Record, 0)
	w.records.ForEach(func(m proto.Message) {
		wrapped = append(wrapped, WrapRecord(w.records, m.(*pb.Record)))
	})
	return wrapped
}

// AllBut returns a wrapped list of records in the current storage
// but the one specified.
func (w Records) AllBut(exclude Record) []Record {
	wrapped := make([]Record, 0)
	w.records.ForEach(func(m proto.Message) {
		record := m.(*pb.Record)
		if record.Id != exclude.record.Id {
			wrapped = append(wrapped, WrapRecord(w.records, record))
		}
	})
	return wrapped
}
