package storage

import (
	pb "github.com/evilsocket/sum/proto"

	"github.com/golang/protobuf/proto"
)

// Records is a thread safe data structure used to
// index and manage records holding vectors and
// any meta data associated to them.
type Records struct {
	*Index
}

// LoadRecords loads and indexes raw protobuf records from
// the data files found in a given path.
func LoadRecords(dataPath string) (*Records, error) {
	recs := &Records{
		Index: NewIndex(dataPath),
	}

	recs.Maker(func() proto.Message { return new(pb.Record) })
	recs.Hasher(func(m proto.Message) uint64 { return m.(*pb.Record).Id })
	recs.Marker(func(m proto.Message, mark uint64) { m.(*pb.Record).Id = mark })
	recs.Copier(func(mold proto.Message, mnew proto.Message) error {
		old := mold.(*pb.Record)
		new := mnew.(*pb.Record)
		if new.Meta != nil {
			old.Meta = new.Meta
		}
		if new.Data != nil {
			old.Data = new.Data
		}
		return nil
	})

	if err := recs.Load(); err != nil {
		return nil, err
	}

	return recs, nil
}

func (r *Records) Find(id uint64) *pb.Record {
	if m := r.Index.Find(id); m != nil {
		return m.(*pb.Record)
	}
	return nil
}

func (r *Records) Delete(id uint64) *pb.Record {
	if m := r.Index.Delete(id); m != nil {
		return m.(*pb.Record)
	}
	return nil
}
