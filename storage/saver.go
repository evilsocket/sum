package storage

import (
	"fmt"
	"io/ioutil"

	"github.com/golang/protobuf/proto"

	"github.com/satori/go.uuid"
)

func NewID() string {
	return uuid.Must(uuid.NewV4()).String()
}

func Flush(m proto.Message, fileName string) error {
	data, err := proto.Marshal(m)
	if err != nil {
		return fmt.Errorf("Error while serializing message to %s: %s", fileName, err)
	} else if err = ioutil.WriteFile(fileName, data, 0755); err != nil {
		return fmt.Errorf("Error while saving message to %s: %s", fileName, err)
	}
	return nil
}
