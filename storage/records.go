package storage

import (
	pb "github.com/evilsocket/sum/proto"
)

// Records is specialized version of a storage.Index
// used to map, store and persist pb.Record objects.
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

// Find returns the instance of a stored pb.Record given its
// identifier or nil if the object can not be found.
func (r *Records) Find(id uint64) *pb.Record {
	if m := r.Index.Find(id); m != nil {
		return m.(*pb.Record)
	}
	return nil
}

// Delete removes a stored pb.Record from the index given its identifier,
// it will return the removed object itself if found, or nil.
func (r *Records) Delete(id uint64) *pb.Record {
	if m := r.Index.Delete(id); m != nil {
		return m.(*pb.Record)
	}
	return nil
}
