package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/proto"
)

const (
	DatFileExt = ".dat"
)

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

		fileID := strings.Replace(fileName, DatFileExt, "", -1)
		loadable[fileID] = filepath.Join(dataPath, fileName)
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
