package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"github.com/evilsocket/islazy/fs"
	"github.com/evilsocket/islazy/log"
	pb "github.com/evilsocket/sum/proto"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func doImport(cli pb.SumServiceClient, fileName string, batchSize int) {
	ext := filepath.Ext(fileName)
	metaFileName := strings.Replace(fileName, ext, ".json", -1)
	metaMapping := make(map[string][]map[string]interface{})

	if fs.Exists(metaFileName) {
		log.Info("reading meta file %s ...", filepath.Base(metaFileName))
		raw, err := ioutil.ReadFile(metaFileName)
		if err != nil {
			die("%v\n", err)
		}

		if err := json.Unmarshal(raw, &metaMapping); err != nil {
			die("%v\n", err)
		}
	}

	log.Info("importing %s ...", fileName)

	dataStartIdx := 0
	dataSize := 0
	imported := 0

	if len(metaMapping["meta"]) > 0 {
		for _, mapping := range metaMapping["meta"] {
			col := int(mapping["column"].(float64))
			if col >= dataStartIdx {
				dataStartIdx = col + 1
			}
		}
		log.Info("data starts at %d", dataStartIdx)
		log.Info("meta mapping: %+v", metaMapping["meta"])
	}

	fp, err := os.Open(fileName)
	if err != nil {
		die("%v\n", err)
	}
	defer fp.Close()
	reader := csv.NewReader(bufio.NewReader(fp))

	batch := pb.Records{
		Records: make([] *pb.Record, batchSize),
	}
	batchIdx := 0
	batchLast := batchSize - 1

	for {
		parts, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			die("%v\n", err)
		}

		// compute number of data columns just once
		totParts := len(parts)
		if dataSize == 0 {
			dataSize = totParts - dataStartIdx
		}

		meta := make(map[string]string)
		data := make([]float32, dataSize)

		// do we need to read metas?
		if dataStartIdx >= 0 {
			for _, mapping := range metaMapping["meta"] {
				col := int(mapping["column"].(float64))
				lbl := mapping["label"].(string)
				meta[lbl] = parts[col]
			}
		}

		partsIdx := dataStartIdx
		dataIdx := 0
		for ; partsIdx < totParts; {
			v := parts[partsIdx]
			if f, err := strconv.ParseFloat(v, 32); err != nil {
				die("%v\n", err)
			} else {
				data[dataIdx] = float32(f)
			}

			partsIdx++
			dataIdx++
		}

		batch.Records[batchIdx] = &pb.Record{
			Meta: meta,
			Data: data,
		}

		if batchIdx == batchLast {
			resp, err := cli.CreateRecords(context.TODO(), &batch)
			if err != nil {
				die("%v\n", err)
			} else if resp.Success == false {
				die("%v\n", resp.Msg)
			}

			imported += batchSize
			batchIdx = 0

			log.Info("imported %d records ...", imported)
		} else {
			batchIdx++
		}
	}

	if batchIdx >= 0 {
		resp, err := cli.CreateRecords(context.TODO(), &batch)
		if err != nil {
			die("%v\n", err)
		} else if resp.Success == false {
			die("%v\n", resp.Msg)
		}

		imported += batchSize
		batchIdx = 0

		log.Info("imported %d records ...", imported)
	}
}
