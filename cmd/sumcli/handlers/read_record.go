package handlers

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	pb "github.com/evilsocket/sum/proto"

	"github.com/chzyer/readline"
)

func dataAsString(data []float64, limit int) string {
	tot := len(data)
	num := tot
	if limit > 0 && limit < tot {
		num = limit
	}
	last := num - 1
	strs := make([]string, num)
	for i, f := range data {
		if f == 0.0 {
			strs[i] = "0"

		} else if f == 1.0 {
			strs[i] = "1"

		} else {
			strs[i] = fmt.Sprintf("%f", f)
		}

		if i == last {
			break
		}
	}
	s := strings.Join(strs, ",")
	if num < tot {
		s += " ..."
	}
	return s
}

func metaAsString(meta map[string]string) string {
	keys := []string{}
	for key, _ := range meta {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := []string{}
	for _, key := range keys {
		value := meta[key]
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(parts, " ")
}

func showRecord(rec *pb.Record, dataLimit int) {
	fmt.Printf("id   : %d\n", rec.Id)
	fmt.Printf("data : %s\n", dataAsString(rec.Data, dataLimit))
	fmt.Printf("meta : %s\n", metaAsString(rec.Meta))
}

var readRecordHandler = handler{
	Name:        "READ",
	Mnemonic:    "READ or R <ID>",
	Completer:   readline.PcItem("read"),
	Parser:      regexp.MustCompile(`^(?i)(READ|R)\s+(\d+)$`),
	Description: "Read the data and metadata of a record given its <ID>.",
	Callback: func(cmd string, args []string, reader *readline.Instance, client pb.SumServiceClient) error {
		id, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return err
		}

		resp, err := client.ReadRecord(context.TODO(), &pb.ById{Id: id})
		if err != nil {
			return err
		} else if resp.Success == false {
			return fmt.Errorf("%s", resp.Msg)
		}

		showRecord(resp.Record, 0)

		return nil
	},
}
