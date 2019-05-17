package handlers

import (
	"context"
	"fmt"
	"github.com/evilsocket/islazy/tui"
	pb "github.com/evilsocket/sum/proto"
	"os"
	"time"

	"github.com/chzyer/readline"
	"github.com/dustin/go-humanize"
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
			"pid",
			"name",
			"os/arch",
			"ver",
			"uptime",
			"records",
			"backend",
		}
		rows := [][]string{}

		for _, n := range resp.Nodes {
			back := fmt.Sprintf("%s (%s of %s)", n.Info.Backend, humanize.Bytes(n.Info.BackendUsed), humanize.Bytes(n.Info.BackendSpace))
			row := []string{
				fmt.Sprintf("%d", n.Id),
				fmt.Sprintf("%d", n.Info.Pid),
				n.Name,
				fmt.Sprintf("%s/%s", n.Info.Os, n.Info.Arch),
				n.Info.Version,
				fmt.Sprintf("%s", time.Duration(n.Info.Uptime)*time.Second),
				fmt.Sprintf("%d", n.Info.Records),
				back,
			}
			rows = append(rows, row)
		}

		tui.Table(os.Stdout, columns, rows)

		return nil
	},
}
