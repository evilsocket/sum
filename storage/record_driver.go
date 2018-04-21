package storage

import (
	"github.com/golang/protobuf/proto"

	pb "github.com/evilsocket/sum/proto"
)

// RecordDriver is the specialized implementation of a
// storage.Driver interface, used to access the internal
// fields of pb.Record objects in the index.
type RecordDriver struct {
}

// Make returns a new pb.Record object.
func (d RecordDriver) Make() proto.Message {
	return new(pb.Record)
}

// GetID returns the unique identifier of the pb.Record object.
func (d RecordDriver) GetID(m proto.Message) uint64 {
	return m.(*pb.Record).Id
}

// SetID sets the unique identifier of the pb.Record object.
func (d RecordDriver) SetID(m proto.Message, id uint64) {
	m.(*pb.Record).Id = id
}

// Copy copies the Meta and Data fields, if filled, from the
// source object to the destination one.
func (d RecordDriver) Copy(mdst proto.Message, msrc proto.Message) error {
	dst := mdst.(*pb.Record)
	src := msrc.(*pb.Record)
	if src.Meta != nil {
		dst.Meta = src.Meta
	}
	if src.Data != nil {
		dst.Data = src.Data
	}
	return nil
}
