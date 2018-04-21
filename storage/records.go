package storage

import (
	pb "github.com/evilsocket/sum/proto"
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
		Index: WithDriver(dataPath, RecordDriver{}),
	}

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
