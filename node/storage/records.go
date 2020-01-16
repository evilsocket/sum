package storage

import (
	pb "github.com/evilsocket/sum/proto"
	"github.com/golang/protobuf/proto"
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

func (r *Records) _metaIndexCreate(rec *pb.Record) {
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

func (r *Records) metaIndexCreate(rec *pb.Record) {
	r.Lock()
	defer r.Unlock()

	r._metaIndexCreate(rec)
}

func (r *Records) _metaIndexUpdate(rec *pb.Record) {
	r._metaIndexRemove(rec)
	r._metaIndexCreate(rec)
}

func (r *Records) metaIndexUpdate(rec *pb.Record) {
	r.metaIndexRemove(rec)
	r.metaIndexCreate(rec)
}

func (r *Records) _metaIndexRemove(rec *pb.Record) {
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

func (r *Records) metaIndexRemove(rec *pb.Record) {
	r.Lock()
	defer r.Unlock()

	r._metaIndexRemove(rec)
}

func (r *Records) metaIndexFind(meta, val string) []uint64 {
	r.RLock()
	defer r.RUnlock()

	if metaIdx, found := r.metaBy[meta]; !found {
		return nil
	} else if bucket, found := metaIdx[val]; !found {
		return nil
	} else {
		return bucket[:]
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
	var records []*pb.Record
	for _, recId := range r.metaIndexFind(meta, val) {
		if m := r.Index.Find(recId); m != nil {
			records = append(records, m.(*pb.Record))
		}
	}

	return records
}

func (r *Records) _createUsing(record *pb.Record, creator func(proto.Message) error) error {
	// if the shape was not provide, it is 1d
	if record.Shape == nil {
		record.Shape = []uint64{uint64(len(record.Data))}
	}

	if err := creator(record); err != nil {
		return err
	} else if len(record.Meta) > 0 {
		// create the meta index for this new record
		r.metaIndexCreate(record)
	}
	return nil
}

func (r *Records) _createManyUsing(records []*pb.Record, creator func([]proto.Message) error) (err error) {
	if len(records) == 0 {
		return
	}

	haveMeta := false
	messages := make([]proto.Message, len(records))
	for i, record := range records {
		// if the shape was not provide, it is 1d
		if record.Shape == nil {
			record.Shape = []uint64{uint64(len(record.Data))}
		}
		if !haveMeta && len(record.Meta) > 0 {
			haveMeta = true
		}
		messages[i] = record
	}

	if err = creator(messages); err != nil {
		return
	} else if !haveMeta {
		return
	}

	r.Lock()
	defer r.Unlock()

	for _, record := range records {
		if len(record.Meta) > 0 {
			r._metaIndexCreate(record)
		}
	}

	return
}

func (r *Records) Create(record *pb.Record) error {
	return r._createUsing(record, r.Index.Create)
}

func (r *Records) CreateMany(records *pb.Records) (err error) {
	return r._createManyUsing(records.Records, r.Index.CreateMany)
}

func (r *Records) CreateWithId(record *pb.Record) error {
	return r._createUsing(record, r.Index.CreateWithId)
}

func (r *Records) CreateManyWIthId(records []*pb.Record) error {
	return r._createManyUsing(records, r.Index.CreateManyWIthId)
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

func (r *Records) DeleteMany(ids []uint64) []*pb.Record {
	res := make([]*pb.Record, 0, len(ids))

	deleted := r.Index.DeleteMany(ids)

	if len(deleted) == 0 {
		return res
	}

	r.Lock()
	defer r.Unlock()

	for _, record := range deleted {
		rec := record.(*pb.Record)
		if len(rec.Meta) > 0 {
			r._metaIndexRemove(rec)
		}
		res = append(res, rec)
	}

	return res
}
