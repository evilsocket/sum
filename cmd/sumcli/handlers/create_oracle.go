package handlers

import (
	"context"
	"fmt"
	"io/ioutil"
	"regexp"

	pb "github.com/evilsocket/sum/proto"

	"github.com/chzyer/readline"
)

var createOracleHandler = handler{
	Name:        "OCREATE",
	Mnemonic:    "OCREATE or OC <NAME> <FILEPATH>",
	Completer:   readline.PcItem("ocreate"),
	Parser:      regexp.MustCompile(`^(?i)(OCREATE|OC)\s+([^\s]+)\s+(.+)$`),
	Description: "Create an oracle with a given <NAME> and the code from a given <FILEAPATH>.",
	Callback: func(cmd string, args []string, reader *readline.Instance, client pb.SumServiceClient) error {
		name := args[0]
		path := args[1]
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		resp, err := client.FindOracle(context.TODO(), &pb.ByName{Name: name})
		if err != nil {
			return err
		} else if resp.Success == true {
			return fmt.Errorf("name '%s' already taken by oracle %d", name, resp.Oracle.Id)
		}

		oracle := pb.Oracle{
			Name: name,
			Code: string(data),
		}
		resp, err = client.CreateOracle(context.TODO(), &oracle)
		if err != nil {
			return err
		} else if resp.Success == false {
			return fmt.Errorf("%v", resp.Msg)
		}

		fmt.Printf("oracle created with id %s\n", resp.Msg)

		return nil
	},
}
