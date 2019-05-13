package storage

import (
	pb "github.com/evilsocket/sum/proto"
)

// Oracles is specialized version of a storage.Index
// used to map, store and persist pb.Oracle objects.
type Oracles struct {
	*Index
}

// LoadOracles loads raw protobuf oracles from
// the data files found in a given path.
func LoadOracles(dataPath string) (*Oracles, error) {
	o := &Oracles{
		Index: WithDriver(dataPath, OracleDriver{}),
	}

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
