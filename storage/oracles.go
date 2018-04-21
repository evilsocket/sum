package storage

import (
	pb "github.com/evilsocket/sum/proto"

	"github.com/golang/protobuf/proto"
)

// Oracles is a thread safe data structure used to
// index and manage oracles.
type Oracles struct {
	*Index
}

// LoadOracles loads raw protobuf oracles from
// the data files found in a given path.
func LoadOracles(dataPath string) (*Oracles, error) {
	o := &Oracles{
		Index: NewIndex(dataPath),
	}

	o.Maker(func() proto.Message { return new(pb.Oracle) })
	o.Hasher(func(m proto.Message) uint64 { return m.(*pb.Oracle).Id })
	o.Marker(func(m proto.Message, mark uint64) { m.(*pb.Oracle).Id = mark })
	o.Copier(func(mold proto.Message, mnew proto.Message) error {
		old := mold.(*pb.Oracle)
		new := mnew.(*pb.Oracle)
		old.Name = new.Name
		old.Code = new.Code
		return nil
	})

	if err := o.Load(); err != nil {
		return nil, err
	}

	return o, nil
}

// Find returns a *pb.Oracle object given its identifier,
// or nil if not found.
func (o *Oracles) Find(id uint64) *pb.Oracle {
	if m := o.Index.Find(id); m != nil {
		return m.(*pb.Oracle)
	}
	return nil
}

// Delete removes an oracle from the index given its
// identifier, it returns the deleted raw *pb.Oracle
// object, or nil if not found.
func (o *Oracles) Delete(id uint64) *pb.Oracle {
	if m := o.Index.Delete(id); m != nil {
		return m.(*pb.Oracle)
	}
	return nil
}
