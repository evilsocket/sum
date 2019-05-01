package handlers

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	pb "github.com/evilsocket/sum/proto"

	"github.com/chzyer/readline"
)

var updateRecordHandler = handler{
	Name:        "UPDATE",
	Mnemonic:    "UPDATE or U <ID>",
	Completer:   readline.PcItem("update"),
	Parser:      regexp.MustCompile(`^(?i)(UPDATE|U)\s+(\d+)$`),
	Description: "Update a record given its <ID> with specified elements and metadata.",
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

		fmt.Printf("updating:\n\n")

		showRecord(resp.Record, 10)

		fmt.Println()

		data, err := readData(reader)
		if err != nil {
			return err
		} else if data == nil {
			return nil
		}

		meta, err := readMetas(reader)
		if err != nil {
			return err
		}

		record := pb.Record{
			Id:   id,
			Data: data,
			Meta: meta,
		}

		resp, err = client.UpdateRecord(context.TODO(), &record)
		if err != nil {
			return err
		} else if resp.Success == false {
			return fmt.Errorf("%v", resp.Msg)
		}

		fmt.Printf("record %d successfully updated.\n", id)

		return nil
	},
}
