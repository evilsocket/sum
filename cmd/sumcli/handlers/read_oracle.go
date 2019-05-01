package handlers

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	pb "github.com/evilsocket/sum/proto"

	"github.com/chzyer/readline"
)

func showOracle(o *pb.Oracle) {
	fmt.Printf("id   : %d\n", o.Id)
	fmt.Printf("name : %s\n", o.Name)
	fmt.Printf("\n%s\n", o.Code)
}

var readOracleHandler = handler{
	Name:        "OREAD",
	Mnemonic:    "OREAD or OR <ID>",
	Completer:   readline.PcItem("oread"),
	Parser:      regexp.MustCompile(`^(?i)(OREAD|OR)\s+(\d+)$`),
	Description: "Read an oracle given its <ID>.",
	Callback: func(cmd string, args []string, reader *readline.Instance, client pb.SumServiceClient) error {
		id, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return err
		}

		resp, err := client.ReadOracle(context.TODO(), &pb.ById{Id: id})
		if err != nil {
			return err
		} else if resp.Success == false {
			return fmt.Errorf("%s", resp.Msg)
		}

		showOracle(resp.Oracle)

		return nil
	},
}
