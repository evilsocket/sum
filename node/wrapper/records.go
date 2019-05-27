package wrapper

import (
	"github.com/evilsocket/sum/node/storage"

	pb "github.com/evilsocket/sum/proto"
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

func (_ Records) New(record *pb.Record) *Record {
	return WrapRecord(record)
}

// Find returns a wrapped Record given its identifier.
// If not found, the resulting record will result as null
// (record.IsNull() will be true).
func (w Records) Find(id uint64) *Record {
	return WrapRecord(w.records.Find(id))
}

// All returns a wrapped list of records in the current storage.
func (w Records) All() []*Record {
	wrapped := make([]*Record, 0)
	w.records.ForEach(func(m proto.Message) error {
		wrapped = append(wrapped, WrapRecord(m.(*pb.Record)))
		return nil
	})
	return wrapped
}

// AllBut returns a wrapped list of records in the current storage
// but the one specified.
func (w Records) AllBut(exclude *Record) []*Record {
	wrapped := make([]*Record, 0)
	w.records.ForEach(func(m proto.Message) error {
		record := m.(*pb.Record)
		if record.Id != exclude.record.Id {
			wrapped = append(wrapped, WrapRecord(record))
		}
		return nil
	})
	return wrapped
}

// CreateRecord creates a new record from raw data.
func (w Records) CreateRecord(data []float32) *Record {
	r := new(pb.Record)
	r.Data = data
	return WrapRecord(r)
}
