package wrapper

import (
	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/storage"
)

type Records struct {
	records *storage.Records
}

func ForRecords(records *storage.Records) Records {
	return Records{
		records: records,
	}
}

func (w Records) Find(id string) Record {
	return ForRecord(w.records.Find(id))
}

func (w Records) All() []Record {
	wrapped := make([]Record, 0)
	w.records.ForEach(func(record *pb.Record) {
		wrapped = append(wrapped, ForRecord(record))
	})
	return wrapped
}

func (w Records) AllBut(exclude Record) []Record {
	wrapped := make([]Record, 0)
	w.records.ForEach(func(record *pb.Record) {
		if record.Id != exclude.record.Id {
			wrapped = append(wrapped, ForRecord(record))
		}
	})
	return wrapped
}
