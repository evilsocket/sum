package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/proto"

	"github.com/satori/go.uuid"
)

const (
	DatFileExt = ".dat"
)

func NewID() string {
	return uuid.Must(uuid.NewV4()).String()
}

func ListPath(dataPath string) (string, map[string]string, error) {
	if dataPath, err := filepath.Abs(dataPath); err != nil {
		return "", nil, err
	} else if info, err := os.Stat(dataPath); err != nil {
		return "", nil, err
	} else if info.IsDir() == false {
		return "", nil, fmt.Errorf("%s is not a folder.", dataPath)
	}

	files, err := ioutil.ReadDir(dataPath)
	if err != nil {
		return "", nil, err
	}

	loadable := make(map[string]string)
	for _, file := range files {
		fileName := file.Name()
		fileExt := filepath.Ext(fileName)
		if fileExt != DatFileExt {
			continue
		}

		fileUUID := strings.Replace(fileName, DatFileExt, "", -1)
		if _, err := uuid.FromString(fileUUID); err == nil {
			loadable[fileUUID] = filepath.Join(dataPath, fileName)
		}
	}

	return dataPath, loadable, nil
}

func Load(fileName string, m proto.Message) error {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("Error while reading %s: %s", fileName, err)
	}

	err = proto.Unmarshal(data, m)
	if err != nil {
		return fmt.Errorf("Error while deserializing %s: %s", fileName, err)
	}
	return nil
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
