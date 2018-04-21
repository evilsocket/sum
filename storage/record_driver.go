package storage

import (
	"github.com/golang/protobuf/proto"

	pb "github.com/evilsocket/sum/proto"
)

type RecordDriver struct {
}

func (d RecordDriver) Make() proto.Message {
	return new(pb.Record)
}

func (d RecordDriver) GetId(m proto.Message) uint64 {
	return m.(*pb.Record).Id
}

func (d RecordDriver) SetId(m proto.Message, id uint64) {
	m.(*pb.Record).Id = id
}

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
