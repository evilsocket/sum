package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/evilsocket/sum/master"
	pb "github.com/evilsocket/sum/proto"

	"github.com/evilsocket/islazy/log"
	"github.com/evilsocket/islazy/str"
	"github.com/evilsocket/islazy/tui"
)

var (
	masterAddress = flag.String("address", "localhost:50051", "Address and port to bind the master process to.")
	masterFile    = flag.String("master", "master.json", "Output file to generate node configurations to.")
	certPath      = flag.String("creds", "/etc/sumd/creds/cert.pem", "Path to the cert.pem file to use for TLS based authentication.")
	basePort      = flag.Int("base-port", 1000, "Port to start to bind slave processes to.")
	numNodes      = flag.Int("num-nodes", -1, "Number of slave processes to create or -1 to spawn one per logical CPU.")

	masterConfig = master.Config{}
)

type childOutputWriter struct {
	Cmd   *exec.Cmd
	ID    string
	Error bool
}

func (w childOutputWriter) Write(p []byte) (n int, err error) {
	data := str.Trim(string(p))
	for _, line := range str.SplitBy(data, "\n") {
		log.Raw("[%s] (%d) %s", tui.Bold(w.ID), w.Cmd.Process.Pid, str.Trim(line))
	}
	return len(p), nil
}

func run(id string, name string, arg ...string) error {
	cmd := exec.Command(name, arg...)

	cmd.Stdout = childOutputWriter{ID: id, Cmd: cmd}
	cmd.Stderr = childOutputWriter{ID: id, Cmd: cmd, Error: true}

	return cmd.Run()
}

func checkDatapath(path string) error {
	paths := []string{
		path,
		path + "/oracles",
		path + "/data",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			log.Info("creating %s ...", p)
			if err := os.MkdirAll(p, os.ModePerm); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkClient(addr string) (bool, error) {
	ctx, _ := context.WithTimeout(context.Background(), 500*time.Millisecond)
	creds, err := credentials.NewClientTLSFromFile(*certPath, str.SplitBy(addr, ":")[0])
	if err != nil {
		log.Fatal("failed to create TLS credentials: %v", err)
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
	}

	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		log.Fatal("failed to dial %s: %v", addr, err)
	}
	defer conn.Close()

	client := pb.NewSumServiceClient(conn)

	if _, err = client.Info(ctx, &pb.Empty{}); err == nil {
		return true, nil
	}

	return false, err
}

func waitClient(addr string) error {
	for logged := false; ; {
		if ok, _ := checkClient(addr); ok == true {
			break
		} else if logged == false {
			log.Debug("waiting for %s ...", addr)
			logged = true
		}
	}

	log.Debug("%s ready", addr)

	return nil
}

func init() {
	log.Format = fmt.Sprintf("[%s] (%d) %s", tui.Bold("cluster"), os.Getpid(), log.Format)
	flag.Parse()
}

func main() {

	if *numNodes <= 0 {
		*numNodes = runtime.NumCPU()
	}

	start := *basePort
	end := start + *numNodes

	log.Info("spawning %d nodes from port %d to %d ...", *numNodes, start, end-1)

	masterConfig.Nodes = make([]master.NodeConfig, 0)
	for port := start; port < end; port++ {
		address := fmt.Sprintf("localhost:%d", port)
		dataPath := fmt.Sprintf("/var/lib/sumd_%02d", port-start)

		if err := checkDatapath(dataPath); err != nil {
			panic(err)
		}

		go func(addr string, path string) {
			if err := run(fmt.Sprintf("node %s", addr), "sumd", "--listen", addr, "--datapath", path); err != nil {
				panic(err)
			}
		}(address, dataPath)

		masterConfig.Nodes = append(masterConfig.Nodes, master.NodeConfig{
			Address:  address,
			CertFile: *certPath,
		})
	}

	log.Info("saving %s", *masterFile)
	if js, err := json.Marshal(masterConfig); err != nil {
		panic(err)
	} else if err = ioutil.WriteFile(*masterFile, js, 0644); err != nil {
		panic(err)
	}

	for _, node := range masterConfig.Nodes {
		if err := waitClient(node.Address); err != nil {
			panic(err)
		}
	}

	log.Info("spawing master process ...")
	if err := run(fmt.Sprintf("master %s", *masterAddress), "sumd", "--listen", *masterAddress, "--master", *masterFile); err != nil {
		panic(err)
	}
}
