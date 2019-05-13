package master

import (
	"context"
	pb "github.com/evilsocket/sum/proto"
	. "github.com/stretchr/testify/require"
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
