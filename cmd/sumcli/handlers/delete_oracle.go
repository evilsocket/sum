package handlers

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	pb "github.com/evilsocket/sum/proto"

	"github.com/chzyer/readline"
)

var deleteOracleHandler = handler{
	Name:        "ODELETE",
	Mnemonic:    "ODELETE or OD <ID>",
	Completer:   readline.PcItem("odelete"),
	Parser:      regexp.MustCompile(`^(?i)(ODELETE|OD)\s+(\d+)$`),
	Description: "Delete an oracle given its <ID>.",
	Callback: func(cmd string, args []string, reader *readline.Instance, client pb.SumServiceClient) error {
		id, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return err
		}

		resp, err := client.DeleteOracle(context.TODO(), &pb.ById{Id: id})
		if err != nil {
			return err
		} else if resp.Success == false {
			return fmt.Errorf("%s", resp.Msg)
		}

		fmt.Printf("oracle %d successfully deleted.\n", id)

		return nil
	},
}
