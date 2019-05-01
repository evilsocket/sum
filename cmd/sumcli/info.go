package main

import (
	"context"
	"fmt"
	"os"
	"reflect"

	pb "github.com/evilsocket/sum/proto"

	"github.com/evilsocket/islazy/str"
	"github.com/evilsocket/islazy/tui"

	"github.com/chzyer/readline"
)

var infoHandler = handler{
	Name:        "INFO",
	Mnemonic:    "INFO",
	Completer:   readline.PcItem("info"),
	Description: "Display server information.",
	Callback: func(cmd string, args []string, reader *readline.Instance, client pb.SumServiceClient) error {
		info, err := client.Info(context.TODO(), &pb.Empty{})
		if err != nil {
			return err
		}

		rows := [][]string{}
		fields := reflect.TypeOf(*info)
		values := reflect.ValueOf(*info)

		for i := 0; i < fields.NumField(); i++ {
			if fieldName := str.Comma(fields.Field(i).Tag.Get("json"))[0]; fieldName != "" && fieldName != "-" {
				value := values.Field(i)
				fieldValue := ""

				switch value.Kind() {
				case reflect.String:
					fieldValue = value.String()
				case reflect.Int:
				case reflect.Int32:
				case reflect.Int64:
				case reflect.Uint:
				case reflect.Uint32:
				case reflect.Uint64:
					fieldValue = fmt.Sprintf("%d", value)
				default:
					fieldValue = value.String()
				}

				rows = append(rows, []string{
					fieldName,
					fieldValue,
				})
			}
		}

		tui.Table(os.Stdout, []string{"name", "value"}, rows)

		return nil
	},
}
