package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/evilsocket/sum/cmd/sumcli/handlers"

	pb "github.com/evilsocket/sum/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/evilsocket/islazy/str"

	"github.com/chzyer/readline"
)

const (
	prompt  = "\033[31m»\033[0m "
	history = "/tmp/sumcli.tmp"
)

var (
	serverAddress = flag.String("address", "127.0.0.1:50051", "Server connection string.")
	serverName    = flag.String("name", "localhost", "The server name use to verify the hostname returned by TLS handshake.")
	certPath      = flag.String("cert", "/etc/sumd/creds/cert.pem", "Path to the cert.pem file to use for TLS based authentication.")
	evalString    = flag.String("eval", "", "List of commands to run, divided by a semicolon.")
	maxMsgSize    = flag.Int("max-msg-size", 50*1024*1024, "Max size of a single GRPC message.")
)

func die(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	os.Exit(1)
}

func main() {
	flag.Parse()

	creds, err := credentials.NewClientTLSFromFile(*certPath, *serverName)
	if err != nil {
		die("failed to create TLS credentials %v\n", err)
	}

	grpc.MaxRecvMsgSize(*maxMsgSize)
	grpc.MaxSendMsgSize(*maxMsgSize)

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(*maxMsgSize),
			grpc.MaxCallSendMsgSize(*maxMsgSize),
		),
	}

	conn, err := grpc.Dial(*serverAddress, opts...)
	if err != nil {
		die("fail to dial: %v\n", err)
	}
	defer conn.Close()

	client := pb.NewSumServiceClient(conn)
	masterClient := pb.NewSumMasterServiceClient(conn)
	reader, err := readline.NewEx(&readline.Config{
		Prompt:          fmt.Sprintf("sumd@%s %s", *serverAddress, prompt),
		HistoryFile:     history,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		AutoComplete:    handlers.Completers,
	})
	if err != nil {
		die("%v\n", err)
	}
	defer reader.Close()

	for _, cmd := range str.SplitBy(*evalString, ";") {
		if err := handlers.Dispatch(cmd, reader, client, masterClient); err != nil {
			fmt.Printf("%s\n", err)
		}
	}

	for {
		if line, err := reader.Readline(); err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		} else {
			for _, cmd := range str.SplitBy(line, ";") {
				if err := handlers.Dispatch(cmd, reader, client, masterClient); err != nil {
					fmt.Printf("%s\n", err)
				}
			}
		}
	}
}
