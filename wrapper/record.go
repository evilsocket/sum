package wrapper

import (
	pb "github.com/evilsocket/sum/proto"
)

type Record struct {
	record *pb.Record
}

func ForRecord(record *pb.Record) Record {
	return Record{
		record: record,
	}
}

func (w Record) IsNull() bool {
	return w.record == nil
}

func (w Record) Is(b Record) bool {
	return w.record.Id == b.record.Id
}

func (w Record) Dot(b Record) float32 {
	dot := float32(0.0)
	for i, va := range w.record.Data {
		vb := b.record.Data[i]
		dot += va * vb
	}
	return dot
}
