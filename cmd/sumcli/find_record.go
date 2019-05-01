package main

import (
	"context"
	"fmt"
	"regexp"

	pb "github.com/evilsocket/sum/proto"

	"github.com/chzyer/readline"
)

var findRecordHandler = handler{
	Name:        "FIND",
	Mnemonic:    "FIND or F <KEY> <VALUE>",
	Completer:   readline.PcItem("find"),
	Parser:      regexp.MustCompile(`^(?i)(FIND|F)\s+([^\s]+)\s+(.+)$`),
	Description: "Find a record by the <KEY> meta data with <VALUE>.",
	Callback: func(cmd string, args []string, reader *readline.Instance, client pb.SumServiceClient) error {
		key := args[0]
		value := args[1]

		query := pb.ByMeta{
			Meta:  key,
			Value: value,
		}
		resp, err := client.FindRecords(context.TODO(), &query)
		if err != nil {
			return err
		} else if resp.Success == false {
			return fmt.Errorf("%s", resp.Msg)
		}

		nrecords := len(resp.Records)
		if nrecords > 0 {
			for _, rec := range resp.Records {
				showRecord(rec, 0)
			}
		} else {
			fmt.Printf("no records found.\n")
		}

		return nil
	},
}
