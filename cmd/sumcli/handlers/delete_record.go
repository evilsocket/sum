package handlers

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	pb "github.com/evilsocket/sum/proto"

	"github.com/chzyer/readline"
)

var deleteRecordHandler = handler{
	Name:        "DELETE",
	Mnemonic:    "DELETE or D <ID>",
	Completer:   readline.PcItem("delete"),
	Parser:      regexp.MustCompile(`^(?i)(DELETE|D)\s+(\d+)$`),
	Description: "Delete a vector given its <ID>.",
	Callback: func(cmd string, args []string, reader *readline.Instance, client pb.SumServiceClient) error {
		id, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return err
		}

		resp, err := client.DeleteRecord(context.TODO(), &pb.ById{Id: id})
		if err != nil {
			return err
		} else if resp.Success == false {
			return fmt.Errorf("%s", resp.Msg)
		}

		fmt.Printf("record %d successfully deleted.\n", id)

		return nil
	},
}
