package storage

import (
	"github.com/golang/protobuf/proto"
)

type Driver interface {
	Make() proto.Message
	GetId(m proto.Message) uint64
	SetId(m proto.Message, id uint64)
	Copy(mdst proto.Message, msrc proto.Message) error
}
