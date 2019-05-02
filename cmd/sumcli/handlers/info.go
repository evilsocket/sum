package handlers

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"

	pb "github.com/evilsocket/sum/proto"

	"github.com/evilsocket/islazy/str"
	"github.com/evilsocket/islazy/tui"

	"github.com/chzyer/readline"
)

func tos(value reflect.Value) string {
	fieldValue := ""
	switch value.Kind() {
	case reflect.String:
		fieldValue = value.String()
	case reflect.Int:
	case reflect.Int8:
	case reflect.Int16:
	case reflect.Int32:
	case reflect.Int64:
		fieldValue = fmt.Sprintf("%d", value.Int())
	case reflect.Uint:
	case reflect.Uint8:
	case reflect.Uint16:
	case reflect.Uint32:
	case reflect.Uint64:
		fieldValue = fmt.Sprintf("%d", value.Uint())
	case reflect.Slice:
		res := []string{}
		for i := 0; i < value.Len(); i++ {
			item := value.Index(i)
			res = append(res, tos(item))
		}
		fieldValue = strings.Join(res, " ")
	default:
		fieldValue = value.String()
	}

	return fieldValue
}

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
				fieldValue := tos(value)

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
