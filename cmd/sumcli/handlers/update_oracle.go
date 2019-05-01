package handlers

import (
	"context"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"

	pb "github.com/evilsocket/sum/proto"

	"github.com/chzyer/readline"
)

var updateOracleHandler = handler{
	Name:        "OUPDATE",
	Mnemonic:    "OUPDATE or OU <ID> <NAME> <FILEPATH>",
	Completer:   readline.PcItem("oupdate"),
	Parser:      regexp.MustCompile(`^(?i)(OUPDATE|OU)\s+(\d+)\s+([^\s]+)\s+(.+)$`),
	Description: "Update an oracle given its <ID> with a new <NAME> and the code from a given <FILEAPATH>.",
	Callback: func(cmd string, args []string, reader *readline.Instance, client pb.SumServiceClient) error {
		id, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return err
		}
		name := args[1]
		path := args[2]
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		// make sure the id is valid
		resp, err := client.ReadOracle(context.TODO(), &pb.ById{Id: id})
		if err != nil {
			return err
		} else if resp.Success == false {
			return fmt.Errorf("%s", resp.Msg)
		}

		// make sure the new name is not taken by another oracle
		resp, err = client.FindOracle(context.TODO(), &pb.ByName{Name: name})
		if err != nil {
			return err
		} else if resp.Success == true && resp.Oracle.Id != id {
			return fmt.Errorf("name '%s' already taken by oracle %d", name, resp.Oracle.Id)
		}

		oracle := pb.Oracle{
			Id:   id,
			Name: name,
			Code: string(data),
		}
		resp, err = client.UpdateOracle(context.TODO(), &oracle)
		if err != nil {
			return err
		} else if resp.Success == false {
			return fmt.Errorf("%v", resp.Msg)
		}

		fmt.Printf("oracle %d succesfully updated\n", id)

		return nil
	},
}
