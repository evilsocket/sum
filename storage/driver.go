package storage

import (
	"github.com/golang/protobuf/proto"
)

// Driver is the generic interface for a module handling
// internal details of a specific protobuf object. It is used
// by the storage.Index in order to access object identifiers
// and internal fields.
type Driver interface {
	// Make must allocate and return a new protobuf object
	// when the index needs to.
	Make() proto.Message
	// GetId must access the protobuf message and return a
	// unique integer identifier that the index will use to
	// map that type of messages.
	GetId(m proto.Message) uint64
	// SetId must access the protobuf message and set its
	// unique integer identifier. This is generally called
	// by instances of storage.Index when a new object is
	// created and a unique id is associated to it for the
	// first time.
	SetId(m proto.Message, id uint64)
	// Copy must copy the contents of the src message into
	// the dst message. An error can be returned to signal
	// the index that something went wrong during the copy.
	Copy(dst proto.Message, src proto.Message) error
}
