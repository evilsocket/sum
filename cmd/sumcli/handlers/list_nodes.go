package handlers

import (
	"context"
	"fmt"
	"github.com/evilsocket/islazy/tui"
	pb "github.com/evilsocket/sum/proto"
	"os"

	"github.com/chzyer/readline"
)

var listNodesHandler = handler{
	Name:        "NLIST",
	Mnemonic:    "NLIST or NL",
	Completer:   readline.PcItem("nlist"),
	Description: "List attached nodes and their stats.",
	MasterCallback: func(cmd string, args []string, reader *readline.Instance, client pb.SumMasterServiceClient) error {
		resp, err := client.ListNodes(context.TODO(), &pb.Empty{})

		if err != nil {
			return err
		} else if !resp.Success {
			return fmt.Errorf("cannot create node: %s", resp.Msg)
		}

		columns := []string{
			"id",
			"name",
			"records",
			"mem",
		}
		rows := [][]string{}

		for _, n := range resp.Nodes {
			row := []string{
				fmt.Sprintf("%d", n.Id),
				n.Name,
				fmt.Sprintf("%d", n.Info.Records),
				fmt.Sprintf("%d", n.Info.Alloc),
			}
			rows = append(rows, row)
		}

		tui.Table(os.Stdout, columns, rows)

		return nil
	},
}
