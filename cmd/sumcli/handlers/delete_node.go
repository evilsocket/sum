package handlers

import (
	"context"
	"fmt"
	pb "github.com/evilsocket/sum/proto"
	"regexp"
	"strconv"

	"github.com/chzyer/readline"
)

var deleteNodeHandler = handler{
	Name:        "NDELETE",
	Mnemonic:    "NDELETE or ND <ID>",
	Completer:   readline.PcItem("ndelete"),
	Parser:      regexp.MustCompile(`^(?i)(NDELETE|ND)\s+(\d+)\s*$`),
	Description: "Create a node by its <ID>.",
	MasterCallback: func(cmd string, args []string, reader *readline.Instance, client pb.SumMasterServiceClient) error {
		id, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return err
		}

		resp, err := client.DeleteNode(context.TODO(), &pb.ById{Id: id})

		if err != nil {
			return err
		} else if !resp.Success {
			return fmt.Errorf("cannot create node: %s", resp.Msg)
		}

		fmt.Printf("node %d deleted\n", id)

		return nil
	},
}
