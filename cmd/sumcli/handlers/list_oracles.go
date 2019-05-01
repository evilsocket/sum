package handlers

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"

	pb "github.com/evilsocket/sum/proto"

	"github.com/chzyer/readline"

	"github.com/evilsocket/islazy/tui"
)

var listOraclesHandler = handler{
	Name:        "OLIST",
	Mnemonic:    "OLIST or OL <PAGE> <PER PAGE>",
	Completer:   readline.PcItem("olist"),
	Parser:      regexp.MustCompile(`^(?i)(OLIST|OL)\s+(\d+)\s+(\d+)$`),
	Description: "Show oracles at <PAGE> while including <PER PAGE> elements per page.",
	Callback: func(cmd string, args []string, reader *readline.Instance, client pb.SumServiceClient) error {
		page, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return err
		}
		per_page, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			return err
		}

		req := pb.ListRequest{
			Page:    page,
			PerPage: per_page,
		}
		resp, err := client.ListOracles(context.TODO(), &req)
		if err != nil {
			return err
		}

		columns := []string{
			"id",
			"name",
			"size",
		}
		rows := [][]string{}

		for _, o := range resp.Oracles {
			row := []string{
				fmt.Sprintf("%d", o.Id),
				o.Name,
				fmt.Sprintf("%d", len(o.Code)),
			}
			rows = append(rows, row)
		}

		tui.Table(os.Stdout, columns, rows)

		fmt.Printf("[page %d of %d (%d total oracles)]\n", page, resp.Pages, resp.Total)

		return nil
	},
}
