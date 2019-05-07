package handlers

import (
	"context"
	"fmt"
	pb "github.com/evilsocket/sum/proto"
	"regexp"

	"github.com/chzyer/readline"
)

var createNodeHandler = handler{
	Name:        "NCREATE",
	Mnemonic:    "NCREATE or NC <ADDR> <FILEPATH>",
	Completer:   readline.PcItem("ncreate"),
	Parser:      regexp.MustCompile(`^(?i)(NCREATE|NC)\s+([^\s]+)\s+(.+)$`),
	Description: "Create a node with listening on ADDR with the given certificate FILEPATH.",
	MasterCallback: func(cmd string, args []string, reader *readline.Instance, client pb.SumMasterServiceClient) error {
		addr := args[0]
		path := args[1]

		arg := &pb.ByAddr{Address: addr, CertFile: path}

		resp, err := client.AddNode(context.TODO(), arg)

		if err != nil {
			return err
		} else if !resp.Success {
			return fmt.Errorf("cannot create node: %s", resp.Msg)
		}

		fmt.Printf("node created with id %s\n", resp.Msg)

		return nil
	},
}
