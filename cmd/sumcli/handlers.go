package main

import (
	"fmt"
	"regexp"
	"strings"

	pb "github.com/evilsocket/sum/proto"

	"github.com/chzyer/readline"
)

type handlerCb func(cmd string, args []string, reader *readline.Instance, client pb.SumServiceClient) error

type handler struct {
	Parser      *regexp.Regexp
	Completer   *readline.PrefixCompleter
	Name        string
	Mnemonic    string
	Description string
	Callback    handlerCb
}

var handlers = []handler{}
var completers = (*readline.PrefixCompleter)(nil)

func init() {
	/*
	   TODO:
	   	// oracles CRUD
	   	CreateOracle(ctx context.Context, in *Oracle, opts ...grpc.CallOption) (*OracleResponse, error)
	   	UpdateOracle(ctx context.Context, in *Oracle, opts ...grpc.CallOption) (*OracleResponse, error)
	   	ReadOracle(ctx context.Context, in *ById, opts ...grpc.CallOption) (*OracleResponse, error)
	   	FindOracle(ctx context.Context, in *ByName, opts ...grpc.CallOption) (*OracleResponse, error)
	   	DeleteOracle(ctx context.Context, in *ById, opts ...grpc.CallOption) (*OracleResponse, error)
	   	// execute a call to a oracle given its id
	   	Run(ctx context.Context, in *Call, opts ...grpc.CallOption) (*CallResponse, error)
	*/
	handlers = []handler{
		helpHandler,
		quitHandler,
		infoHandler,
		// records CRUD
		createRecordHandler,
		readRecordHandler,
		updateRecordHandler,
		deleteRecordHandler,
		listRecordsHandler,
		findRecordHandler,
	}

	tmp := []readline.PrefixCompleterInterface{}
	for _, h := range handlers {
		tmp = append(tmp, h.Completer)
	}
	completers = readline.NewPrefixCompleter(tmp...)
}

func dispatchHandler(cmd string, reader *readline.Instance, client pb.SumServiceClient) error {
	for _, handler := range handlers {
		match := false
		args := []string{}

		if handler.Parser != nil {
			if result := handler.Parser.FindStringSubmatch(cmd); result != nil && len(result) == handler.Parser.NumSubexp()+1 {
				cmd = result[1:][0]
				args = result[1:][1:]
				match = true
			}
		} else if strings.EqualFold(handler.Name, cmd) {
			match = true
		}

		if match {
			return handler.Callback(cmd, args, reader, client)
		}
	}

	return fmt.Errorf("command not found: %s", cmd)
}
