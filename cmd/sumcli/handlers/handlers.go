package handlers

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

var Handlers = []handler{}
var Completers = (*readline.PrefixCompleter)(nil)

func init() {
	Handlers = []handler{
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
		// oracles CRUD and execution
		createOracleHandler,
		readOracleHandler,
		updateOracleHandler,
		deleteOracleHandler,
		findOracleHandler,
		listOraclesHandler,
		callOracleHandler,
	}

	tmp := []readline.PrefixCompleterInterface{}
	for _, h := range Handlers {
		if h.Completer != nil {
			tmp = append(tmp, h.Completer)
		}
	}
	Completers = readline.NewPrefixCompleter(tmp...)
}

func Dispatch(cmd string, reader *readline.Instance, client pb.SumServiceClient) error {
	for _, handler := range Handlers {
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
