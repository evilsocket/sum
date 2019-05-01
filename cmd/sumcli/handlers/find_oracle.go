package handlers

import (
	"context"
	"fmt"
	"regexp"

	pb "github.com/evilsocket/sum/proto"

	"github.com/chzyer/readline"
)

var findOracleHandler = handler{
	Name:        "OFIND",
	Mnemonic:    "OFIND or OF <NAME>",
	Completer:   readline.PcItem("ofind"),
	Parser:      regexp.MustCompile(`^(?i)(OFIND|OF)\s+(.+)$`),
	Description: "Find an oracle given its <NAME>.",
	Callback: func(cmd string, args []string, reader *readline.Instance, client pb.SumServiceClient) error {
		resp, err := client.FindOracle(context.TODO(), &pb.ByName{Name: args[0]})
		if err != nil {
			return err
		} else if resp.Success == false {
			return fmt.Errorf("%s", resp.Msg)
		}

		showOracle(resp.Oracle)

		return nil
	},
}
