package storage

import (
	"github.com/golang/protobuf/proto"

	pb "github.com/evilsocket/sum/proto"
)

// OracleDriver is the specialized implementation of a
// storage.Driver interface, used to access the internal
// fields of pb.Oracle objects in the index.
type OracleDriver struct {
}

// Make returns a new pb.Oracle object.
func (d OracleDriver) Make() proto.Message {
	return new(pb.Oracle)
}

// GetID returns the unique identifier of the pb.Oracle object.
func (d OracleDriver) GetID(m proto.Message) uint64 {
	return m.(*pb.Oracle).Id
}

// SetID sets the unique identifier of the pb.Oracle object.
func (d OracleDriver) SetID(m proto.Message, id uint64) {
	m.(*pb.Oracle).Id = id
}

// Copy copies the Name and Code fields from the source
// object to the destination one.
func (d OracleDriver) Copy(mdst proto.Message, msrc proto.Message) error {
	dst := mdst.(*pb.Oracle)
	src := msrc.(*pb.Oracle)
	dst.Name = src.Name
	dst.Code = src.Code
	return nil
}
