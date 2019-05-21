package master

import (
	"context"
	pb "github.com/evilsocket/sum/proto"
	. "github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveConfig(t *testing.T) {
	tmpDir, err := setupEmptyTmpFolder()
	Nil(t, err)
	defer os.RemoveAll(tmpDir)

	server, ms := spawnOrchestrator(t, 12346, "")
	defer server.Stop()

	ms.configFile = filepath.Join(tmpDir, "cfg.json")
	ms.updateConfig()

	FileExists(t, ms.configFile)

	cfg, err := LoadConfig(ms.configFile)
	Nil(t, err)
	Empty(t, cfg.Nodes)

	node, _ := spawnNode(t, 12345, tmpDir)
	defer node.Stop()

	resp, err := ms.AddNode(context.Background(), &pb.ByAddr{Address: "localhost:12345", CertFile: ""})
	Nil(t, err)
	True(t, resp.Success)

	// wait for writeback routing to finish
	time.Sleep(200 * time.Millisecond)

	cfg, err = LoadConfig(ms.configFile)
	Nil(t, err)
	Equal(t, 1, len(cfg.Nodes))

	nc := cfg.Nodes[0]
	Equal(t, "", nc.CertFile)
	Equal(t, "localhost:12345", nc.Address)

	resp, err = ms.DeleteNode(context.Background(), &pb.ById{Id: 1})
	Nil(t, err)
	True(t, resp.Success)

	// wait for writeback routing to finish
	time.Sleep(200 * time.Millisecond)

	cfg, err = LoadConfig(ms.configFile)
	Nil(t, err)
	Empty(t, cfg.Nodes)
}

func TestBadConfig(t *testing.T) {
	_, err := LoadConfig("/not/found")
	Error(t, err)

	tmpfile, err := ioutil.TempFile("", "")
	NoError(t, err)

	err = ioutil.WriteFile(tmpfile.Name(), []byte("not a json thingy"), 0644)
	NoError(t, err)

	_, err = LoadConfig(tmpfile.Name())
	Error(t, err)
}
