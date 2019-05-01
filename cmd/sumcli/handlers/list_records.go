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

var listRecordsHandler = handler{
	Name:        "LIST",
	Mnemonic:    "LIST or L <PAGE> <PER PAGE>",
	Completer:   readline.PcItem("list"),
	Parser:      regexp.MustCompile(`^(?i)(LIST|L)\s+(\d+)\s+(\d+)$`),
	Description: "Show records at <PAGE> while including <PER PAGE> elements per page.",
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
		resp, err := client.ListRecords(context.TODO(), &req)
		if err != nil {
			return err
		}

		columns := []string{
			"id",
			"size",
			"data",
			"meta",
		}
		rows := [][]string{}

		for _, r := range resp.Records {
			row := []string{
				fmt.Sprintf("%d", r.Id),
				fmt.Sprintf("%d", len(r.Data)),
				dataAsString(r.Data, 10),
				metaAsString(r.Meta),
			}
			rows = append(rows, row)
		}

		tui.Table(os.Stdout, columns, rows)

		fmt.Printf("[page %d of %d (%d total records)]\n", page, resp.Pages, resp.Total)

		return nil
	},
}
