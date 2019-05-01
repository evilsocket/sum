package handlers

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io/ioutil"
	"regexp"

	pb "github.com/evilsocket/sum/proto"

	"github.com/chzyer/readline"

	"github.com/evilsocket/islazy/str"
)

var callOracleHandler = handler{
	Name:        "CALL",
	Mnemonic:    "<NAME>(<ARGUMENTS>)",
	Completer:   nil,
	Parser:      regexp.MustCompile(`^(?i)([^\(]+)\(([^\)]*)\)$`),
	Description: "Call the oracle <NAME> with the specified <ARGUMENTS>.",
	Callback: func(cmd string, args []string, reader *readline.Instance, client pb.SumServiceClient) error {
		resp, err := client.FindOracle(context.TODO(), &pb.ByName{Name: cmd})
		if err != nil {
			return err
		} else if resp.Success == false {
			return fmt.Errorf("%s", resp.Msg)
		}

		call := pb.Call{
			OracleId: resp.Oracle.Id,
			Args:     str.Comma(args[0]),
		}
		cresp, err := client.Run(context.TODO(), &call)
		if err != nil {
			return err
		} else if cresp.Success == false {
			return fmt.Errorf("%s", cresp.Msg)
		}

		data := cresp.Data.Payload
		if cresp.Data.Compressed {
			gr, err := gzip.NewReader(bytes.NewBuffer(data))
			defer gr.Close()
			data, err = ioutil.ReadAll(gr)
			if err != nil {
				return err
			}
		}

		fmt.Printf("%s\n", string(data))

		return nil
	},
}
