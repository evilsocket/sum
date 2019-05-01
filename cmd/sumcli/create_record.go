package main

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"

	pb "github.com/evilsocket/sum/proto"

	"github.com/chzyer/readline"

	"github.com/evilsocket/islazy/str"
)

func readWithPrompt(reader *readline.Instance, prompt string) (string, error) {
	backP := reader.Config.Prompt
	backH := reader.Config.HistoryFile

	reader.SetPrompt(prompt)
	reader.SetHistoryPath("")

	defer func() {
		reader.SetPrompt(backP)
		reader.SetHistoryPath(backH)
	}()

	line, err := reader.Readline()
	if err == readline.ErrInterrupt || err == io.EOF || len(line) == 0 {
		return "", nil
	} else if err != nil {
		return "", err
	}

	return line, nil
}

func readData(reader *readline.Instance) ([]float64, error) {
	raw, err := readWithPrompt(reader, "comma separated values> ")
	if err != nil {
		return nil, err
	} else if raw == "" {
		return nil, nil
	}

	values := str.Comma(raw)
	if len(values) == 0 {
		return nil, fmt.Errorf("could not create empty vector")
	}

	data := make([]float64, len(values))
	for i, v := range values {
		if f, err := strconv.ParseFloat(v, 64); err != nil {
			return nil, err
		} else {
			data[i] = f
		}
	}

	return data, nil
}

func readMetas(reader *readline.Instance) (map[string]string, error) {
	meta := make(map[string]string)

	for {
		raw, err := readWithPrompt(reader, "meta key:value (blank to stop)> ")
		if err != nil {
			return nil, err
		} else if raw == "" {
			break
		}

		values := str.SplitBy(raw, ":")
		nvalues := len(values)
		if nvalues == 0 {
			return nil, fmt.Errorf("could not create empty metadata")
		} else if nvalues != 2 {
			return nil, fmt.Errorf("could not parse '%s' as 'key: value'", raw)
		}

		meta[values[0]] = values[1]
	}

	return meta, nil
}

var createRecordHandler = handler{
	Name:        "CREATE",
	Mnemonic:    "CREATE or C",
	Completer:   readline.PcItem("create"),
	Parser:      regexp.MustCompile(`^(?i)(CREATE|C)$`),
	Description: "Create a new record with specified elements and metadata.",
	Callback: func(cmd string, args []string, reader *readline.Instance, client pb.SumServiceClient) error {
		data, err := readData(reader)
		if err != nil {
			return err
		} else if data == nil {
			return nil
		}

		meta, err := readMetas(reader)
		if err != nil {
			return err
		}

		record := pb.Record{
			Data: data,
			Meta: meta,
		}

		resp, err := client.CreateRecord(context.TODO(), &record)
		if err != nil {
			return err
		} else if resp.Success == false {
			return fmt.Errorf("%v", resp.Msg)
		}

		fmt.Printf("record created with id %s\n", resp.Msg)

		return nil
	},
}
