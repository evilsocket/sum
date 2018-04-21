package storage

import (
	"github.com/golang/protobuf/proto"

	pb "github.com/evilsocket/sum/proto"
)

type OracleDriver struct {
}

func (d OracleDriver) Make() proto.Message {
	return new(pb.Oracle)
}

func (d OracleDriver) GetId(m proto.Message) uint64 {
	return m.(*pb.Oracle).Id
}

func (d OracleDriver) SetId(m proto.Message, id uint64) {
	m.(*pb.Oracle).Id = id
}

func (d OracleDriver) Copy(mdst proto.Message, msrc proto.Message) error {
	dst := mdst.(*pb.Oracle)
	src := msrc.(*pb.Oracle)
	dst.Name = src.Name
	dst.Code = src.Code
	return nil
}
