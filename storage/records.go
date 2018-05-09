package storage

import (
	pb "github.com/evilsocket/sum/proto"
)

type metaIndex map[string][]*pb.Record

// Records is specialized version of a storage.Index
// used to map, store and persist pb.Record objects.
type Records struct {
	*Index

	metaBy map[string]metaIndex
}

// LoadRecords loads and indexes raw protobuf records from
// the data files found in a given path.
func LoadRecords(dataPath string) (*Records, error) {
	recs := &Records{
		Index:  WithDriver(dataPath, RecordDriver{}),
		metaBy: make(map[string]metaIndex),
	}

	if err := recs.Load(); err != nil {
		return nil, err
	}

	for _, m := range recs.index {
		recs.metaIndexCreate(m.(*pb.Record))
	}

	return recs, nil
}

func (r *Records) metaIndexCreate(rec *pb.Record) {
	r.Lock()
	defer r.Unlock()

	for key, val := range rec.Meta {
		// create the index by this key if not there already
		if metaIdx, found := r.metaBy[key]; !found {
			r.metaBy[key] = metaIndex{
				val: []*pb.Record{rec},
			}
		} else {
			metaIdx[val] = append(metaIdx[val], rec)
		}
	}
}

func (r *Records) metaIndexRemove(rec *pb.Record) {
	r.Lock()
	defer r.Unlock()

	for key, val := range rec.Meta {
		// find the bucket for this meta
		if metaIdx, found := r.metaBy[key]; !found {
			// find the bucket by value
			bucket := metaIdx[val]
			for i, elem := range bucket {
				// find the record by id
				if elem.Id == rec.Id {
					// remove it
					metaIdx[val] = append(bucket[:i], bucket[i+1:]...)
					break
				}
			}
		}
	}
}

// Find returns the instance of a stored pb.Record given its
// identifier or nil if the object can not be found.
func (r *Records) Find(id uint64) *pb.Record {
	if m := r.Index.Find(id); m != nil {
		return m.(*pb.Record)
	}
	return nil
}

// FindBy returns the list of pb.Record objects
// indexed by a specific meta value.
func (r *Records) FindBy(meta string, val string) []*pb.Record {
	r.RLock()
	defer r.RUnlock()

	metaIdx, found := r.metaBy[meta]
	if !found {
		return nil
	}

	indexed, found := metaIdx[val]
	if !found {
		return []*pb.Record{}
	}
	return indexed
}

// Delete removes a stored pb.Record from the index given its identifier,
// it will return the removed object itself if found, or nil.
func (r *Records) Delete(id uint64) *pb.Record {
	if m := r.Index.Delete(id); m != nil {
		rec := m.(*pb.Record)
		// remove the record from the meta index
		r.metaIndexRemove(rec)
		return rec
	}
	return nil
}
