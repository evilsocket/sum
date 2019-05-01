package handlers

import (
	"os"
	"regexp"

	pb "github.com/evilsocket/sum/proto"

	"github.com/chzyer/readline"
)

var quitHandler = handler{
	Name:        "QUIT",
	Mnemonic:    "QUIT, Q or EXIT",
	Completer:   readline.PcItem("quit"),
	Parser:      regexp.MustCompile(`^(?i)(QUIT|Q|EXIT)$`),
	Description: "Exit the client.",
	Callback: func(cmd string, args []string, reader *readline.Instance, client pb.SumServiceClient) error {
		os.Exit(0)
		return nil
	},
}
