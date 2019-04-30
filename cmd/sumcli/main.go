package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	pb "github.com/evilsocket/sum/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/evilsocket/islazy/str"
	"github.com/evilsocket/islazy/tui"
)

var (
	serverAddress = flag.String("address", "127.0.0.1:50051", "Server connection string.")
	serverName    = flag.String("name", "localhost", "The server name use to verify the hostname returned by TLS handshake.")
	certPath      = flag.String("cert", "/etc/sumd/creds/cert.pem", "Path to the cert.pem file to use for TLS based authentication.")
	evalString    = flag.String("eval", "info", "List of commands to run, divided by a semicolon.")

	client = (pb.SumServiceClient)(nil)
)

type handlerCb func(cmd string, args []string) error

type handler struct {
	Parser      *regexp.Regexp
	Name        string
	Mnemonic    string
	Description string
	Handler     handlerCb
}

/*
TODO:

	// vectors CRUD
	CreateRecord(ctx context.Context, in *Record, opts ...grpc.CallOption) (*RecordResponse, error)
	UpdateRecord(ctx context.Context, in *Record, opts ...grpc.CallOption) (*RecordResponse, error)
	ReadRecord(ctx context.Context, in *ById, opts ...grpc.CallOption) (*RecordResponse, error)
	ListRecords(ctx context.Context, in *ListRequest, opts ...grpc.CallOption) (*RecordListResponse, error)
	DeleteRecord(ctx context.Context, in *ById, opts ...grpc.CallOption) (*RecordResponse, error)
	// find a vector given a meta name and value to filter for
	FindRecords(ctx context.Context, in *ByMeta, opts ...grpc.CallOption) (*FindResponse, error)
	// oracles CRUD
	CreateOracle(ctx context.Context, in *Oracle, opts ...grpc.CallOption) (*OracleResponse, error)
	UpdateOracle(ctx context.Context, in *Oracle, opts ...grpc.CallOption) (*OracleResponse, error)
	ReadOracle(ctx context.Context, in *ById, opts ...grpc.CallOption) (*OracleResponse, error)
	FindOracle(ctx context.Context, in *ByName, opts ...grpc.CallOption) (*OracleResponse, error)
	DeleteOracle(ctx context.Context, in *ById, opts ...grpc.CallOption) (*OracleResponse, error)
	// execute a call to a oracle given its id
	Run(ctx context.Context, in *Call, opts ...grpc.CallOption) (*CallResponse, error)
*/

var handlers = []handler{
	handler{Name: "INFO", Mnemonic: "INFO", Description: "Bla bla", Handler: func(cmd string, args []string) error {
		info, err := client.Info(context.TODO(), &pb.Empty{})
		if err != nil {
			return err
		}

		rows := [][]string{
			[]string{"server", *serverAddress},
			[]string{"name", *serverName},
			[]string{"certificate", *certPath},
			[]string{"version", info.Version},
			[]string{"uptime", fmt.Sprintf("%s", time.Duration(info.Uptime)*time.Second)},
			[]string{"pid", fmt.Sprintf("%d", info.Pid)},
			[]string{"uid", fmt.Sprintf("%d", info.Uid)},
			[]string{"argv", fmt.Sprintf("%s", info.Argv)},
			[]string{"records", fmt.Sprintf("%d", info.Records)},
			[]string{"oracles", fmt.Sprintf("%d", info.Oracles)},
		}

		tui.Table(os.Stdout, []string{"name", "value"}, rows)

		return nil
	}},
}

func findHandlerFor(cmd string) (int, []string) {
	for idx, handler := range handlers {
		if handler.Parser != nil {

		} else if strings.EqualFold(handler.Name, cmd) {
			return idx, nil
		}
	}
	return -1, nil
}

func main() {
	flag.Parse()

	creds, err := credentials.NewClientTLSFromFile(*certPath, *serverName)
	if err != nil {
		fmt.Printf("failed to create TLS credentials %v\n", err)
		os.Exit(1)
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
	}

	conn, err := grpc.Dial(*serverAddress, opts...)
	if err != nil {
		fmt.Printf("fail to dial: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	client = pb.NewSumServiceClient(conn)

	for _, cmd := range str.SplitBy(*evalString, ";") {
		if idx, args := findHandlerFor(cmd); idx == -1 {
			fmt.Printf("command not found: %s\n", cmd)
		} else if err := handlers[idx].Handler(cmd, args); err != nil {
			fmt.Printf("error while running command '%s': %v\n", cmd, err)
		}
	}
}
