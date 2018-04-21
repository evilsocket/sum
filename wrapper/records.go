package wrapper

import (
	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/storage"
)

// Records is an object that wraps *storage.Records in
// order to give access to those records to oracles during
// execution.
type Records struct {
	records *storage.Records
}

// ForRecords creates a Records wrapper around a *storage.Records object.
func ForRecords(records *storage.Records) Records {
	return Records{
		records: records,
	}
}

// Find returns a wrapped Record given its identifier.
// If not found, the resulting record will result as null
// (record.IsNull() will be true).
func (w Records) Find(id uint64) Record {
	return ForRecord(w.records, w.records.Find(id))
}

// All returns a wrapped list of records in the current storage.
func (w Records) All() []Record {
	wrapped := make([]Record, 0)
	w.records.ForEach(func(record *pb.Record) {
		wrapped = append(wrapped, ForRecord(w.records, record))
	})
	return wrapped
}

// AllBut returns a wrapped list of records in the current storage
// but the one specified.
func (w Records) AllBut(exclude Record) []Record {
	wrapped := make([]Record, 0)
	w.records.ForEach(func(record *pb.Record) {
		if record.Id != exclude.record.Id {
			wrapped = append(wrapped, ForRecord(w.records, record))
		}
	})
	return wrapped
}
