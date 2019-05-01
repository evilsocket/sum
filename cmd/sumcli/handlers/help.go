package handlers

import (
	"os"

	pb "github.com/evilsocket/sum/proto"

	"github.com/evilsocket/islazy/tui"

	"github.com/chzyer/readline"
)

var helpHandler = handler{
	Name:        "HELP",
	Mnemonic:    "HELP",
	Completer:   readline.PcItem("help"),
	Description: "Show the available client commands and their descriptions.",
	Callback: func(cmd string, args []string, reader *readline.Instance, client pb.SumServiceClient) error {
		rows := [][]string{}

		for _, h := range Handlers {
			rows = append(rows, []string{h.Mnemonic, h.Description})
		}

		tui.Table(os.Stdout, []string{"command", "description"}, rows)

		return nil
	},
}
