package storage

import (
	pb "github.com/evilsocket/sum/proto"
)

type metaIndex map[string][]uint64

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
				val: []uint64{rec.Id},
			}
		} else {
			metaIdx[val] = append(metaIdx[val], rec.Id)
		}
	}
}

func (r *Records) metaIndexUpdate(rec *pb.Record) {
	r.metaIndexRemove(rec)
	r.metaIndexCreate(rec)
}

func (r *Records) metaIndexRemove(rec *pb.Record) {
	r.Lock()
	defer r.Unlock()

	for key, val := range rec.Meta {
		// find the bucket for this meta
		if metaIdx, found := r.metaBy[key]; !found {
			// find the bucket by value
			bucket := metaIdx[val]
			for i, elemID := range bucket {
				// find the record by id
				if elemID == rec.Id {
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

	records := []*pb.Record{}
	if bucket, found := metaIdx[val]; found {
		for _, recID := range bucket {
			m := r.Index.Find(recID)
			if m != nil {
				records = append(records, m.(*pb.Record))
			}
		}
	}

	return records
}

func (r *Records) Create(record *pb.Record) error {
	// if the shape was not provide, it is 1d
	if record.Shape == nil {
		record.Shape = []uint64{uint64(len(record.Data))}
	}

	if err := r.Index.Create(record); err != nil {
		return err
	}
	// create the meta index for this new record
	r.metaIndexCreate(record)
	return nil
}

func (r *Records) Update(record *pb.Record) error {
	if err := r.Index.Update(record); err != nil {
		return err
	}
	// update the meta index for this record
	r.metaIndexUpdate(record)
	return nil
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
