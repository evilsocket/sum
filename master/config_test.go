package master

import (
	"context"
	"github.com/evilsocket/islazy/log"
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

	// test error

	tmpfile, err := ioutil.TempFile("", "")
	NoError(t, err)
	defer os.Remove(tmpfile.Name())
	NoError(t, tmpfile.Close())

	restoreLog := func() {
		log.Output = ""
		log.Open()
	}

	log.Output = tmpfile.Name()
	NoError(t, log.Open())
	defer restoreLog()

	ms.configFile = "/suchdirectory/veryunexistingfile"
	ms.updateConfig()

	restoreLog()

	logContent, err := ioutil.ReadFile(tmpfile.Name())

	Contains(t, string(logContent), `cannot save configuration: open /suchdirectory/veryunexistingfile: no such file or directory`)
}

func TestStoreConfig(t *testing.T) {
	cfg := &Config{}

	t.Run("NonExistentFolder", func(t *testing.T) {
		err := StoreConfig(cfg, "/spain/guapaloca")
		Error(t, err)
	})

	t.Run("WithExistentFolderButNoFile", func(t *testing.T) {
		dir, err := ioutil.TempDir("", "")
		NoError(t, err)
		defer os.RemoveAll(dir)

		filePath := filepath.Join(dir, "guapaloca")

		err = StoreConfig(cfg, filePath)
		NoError(t, err)
		FileExists(t, filePath)
	})

	t.Run("WitExistingFile", func(t *testing.T) {
		tmpfile, err := ioutil.TempFile("", "")
		NoError(t, err)
		defer os.Remove(tmpfile.Name())
		NoError(t, tmpfile.Close())

		err = StoreConfig(cfg, tmpfile.Name())
		NoError(t, err)
	})
}

func TestBadConfig(t *testing.T) {
	_, err := LoadConfig("/not/found")
	Error(t, err)

	tmpfile, err := ioutil.TempFile("", "")
	NoError(t, err)
	defer os.Remove(tmpfile.Name())
	NoError(t, tmpfile.Close())

	err = ioutil.WriteFile(tmpfile.Name(), []byte("not a json thingy"), 0644)
	NoError(t, err)

	_, err = LoadConfig(tmpfile.Name())
	Error(t, err)
}

func TestNewServiceFromConfig(t *testing.T) {
	ns, err := setupNetwork(2, 1)
	NoError(t, err)
	defer cleanupNetwork(&ns)

	// save config somewhere

	ms := ns.orchestrators[0].svc

	tmpfile, err := ioutil.TempFile("", "")
	NoError(t, err)
	defer os.Remove(tmpfile.Name())
	NoError(t, tmpfile.Close())

	ms.configFile = tmpfile.Name()
	ms.updateConfig()

	FileExists(t, ms.configFile)

	// stop orchestrator and respawn it

	ns.orchestrators[0].server.Stop()

	// start a new one ( without a listening server, just the service )

	t.Run("WithInvalidFile", func(t *testing.T) {
		_, err = NewServiceFromConfig("/spain/guapaloca", "", "nowhere")
		Error(t, err)
	})

	t.Run("WithBadNodeAddresses", func(t *testing.T) {
		cfgCopy, err := LoadConfig(ms.configFile)
		NoError(t, err)
		cfgCopy.Nodes[0].Address = "besame_mucho"

		tmpfile, err = ioutil.TempFile("", "")
		NoError(t, err)
		defer os.Remove(tmpfile.Name())
		NoError(t, tmpfile.Close())

		NoError(t, StoreConfig(cfgCopy, tmpfile.Name()))

		_, err = NewServiceFromConfig(tmpfile.Name(), "", "nowhere")
		Error(t, err)
	})

	t.Run("Valid", func(t *testing.T) {
		ms1, err := NewServiceFromConfig(ms.configFile, "", "nowhere")
		NoError(t, err)

		resp, err := ms1.ListNodes(context.TODO(), &pb.Empty{})
		NoError(t, err)
		True(t, resp.Success, resp.Msg)
		Equal(t, 2, len(resp.Nodes))
	})
}
