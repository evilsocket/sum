package service

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/storage"
)

const (
	testFolder  = "/tmp/sum.service.test"
	testRecords = 5
)

var (
	testOracle = pb.Oracle{
		Id:   666,
		Name: "findReasonsToLive",
		Code: "function findReasonsToLive(){ return 0; }",
	}
	testRecord = pb.Record{
		Id:   666,
		Data: []float32{0.6, 0.6, 0.6},
		Meta: map[string]string{"666": "666"},
	}
	testCall = pb.Call{
		OracleId: 1,
		Args:     []string{},
	}
)

func unlink(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func decompress(t testing.TB, d *pb.Data) string {
	data := d.Payload
	if d.Compressed == true {
		r, err := gzip.NewReader(bytes.NewBuffer(data))
		if err != nil {
			t.Fatalf("error while decompressing response payload: %s", err)
		}
		defer r.Close()
		if data, err = ioutil.ReadAll(r); err != nil {
			t.Fatalf("error while decompressing response payload: %s", err)
		}
	}
	return string(data)
}

func setup(t testing.TB, withRecords bool, withOracles bool) {
	log.SetOutput(ioutil.Discard)

	// start clean
	teardown(t)

	if err := os.MkdirAll(testFolder, 0755); err != nil {
		t.Fatalf("Error creating %s: %s", testFolder, err)
	}

	if withRecords {
		basePath := filepath.Join(testFolder, dataFolderName)
		if err := os.MkdirAll(basePath, 0755); err != nil {
			t.Fatalf("Error creating folder %s: %s", basePath, err)
		} else if recs, err := storage.LoadRecords(basePath); err != nil {
			t.Fatal(err)
		} else {
			for i := 1; i <= testRecords; i++ {
				if err := recs.Create(&testRecord); err != nil {
					t.Fatalf("Error while creating record: %s", err)
				}
			}
		}
	}

	if withOracles {
		basePath := filepath.Join(testFolder, oraclesFolderName)
		if err := os.MkdirAll(basePath, 0755); err != nil {
			t.Fatalf("Error creating folder %s: %s", basePath, err)
		} else if ors, err := storage.LoadOracles(basePath); err != nil {
			t.Fatal(err)
		} else {
			for i := 1; i <= testOracles; i++ {
				if err := ors.Create(&testOracle); err != nil {
					t.Fatalf("Error creating oracle: %s", err)
				}
			}
		}
	}
}

func teardown(t testing.TB) {
	if err := unlink(testFolder); err != nil {
		if os.IsNotExist(err) == false {
			t.Fatalf("Error deleting %s: %s", testFolder, err)
		}
	}
}

func TestErrCallResponse(t *testing.T) {
	if r := errCallResponse("test %d", 123); r.Success == true {
		t.Fatal("success should be false")
	} else if r.Msg != "test 123" {
		t.Fatalf("unexpected message: %s", r.Msg)
	} else if r.Data != nil {
		t.Fatalf("unexpected data pointer: %v", r.Data)
	}
}

func TestNew(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if svc == nil {
		t.Fatal("expected valid service instance")
	} else if time.Since(svc.started).Seconds() >= 1.0 {
		t.Fatalf("wrong started time: %v", svc.started)
	} else if svc.pid != uint64(os.Getpid()) {
		t.Fatalf("wrong pid: %d", svc.pid)
	} else if svc.uid != uint64(os.Getuid()) {
		t.Fatalf("wrong uid: %d", svc.uid)
	} else if reflect.DeepEqual(svc.argv, os.Args) == false {
		t.Fatalf("wrong args: %v", svc.argv)
	} else if svc.NumRecords() != uint64(testRecords) {
		t.Fatalf("wrong number of records: %d", svc.NumRecords())
	} else if svc.NumOracles() != uint64(testOracles) {
		t.Fatalf("wrong number of oracles: %d", svc.NumOracles())
	}
}

func TestNewWithoutFolders(t *testing.T) {
	defer teardown(t)

	setup(t, false, false)
	if svc, err := New(testFolder); err == nil {
		t.Fatal("expected error")
	} else if svc != nil {
		t.Fatal("expected null service instance")
	}

	setup(t, true, false)
	if svc, err := New(testFolder); err == nil {
		t.Fatal("expected error")
	} else if svc != nil {
		t.Fatal("expected null service instance")
	}
}

func TestInfo(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if info, err := svc.Info(nil, nil); err != nil {
		t.Fatal(err)
	} else if info.Version != Version {
		t.Fatalf("wrong version: %s", info.Version)
	} else if info.Uptime > 1 {
		t.Fatalf("wrong uptime: %d", info.Uptime)
	} else if svc.pid != info.Pid {
		t.Fatalf("wrong pid: %d", info.Pid)
	} else if svc.uid != info.Uid {
		t.Fatalf("wrong uid: %d", info.Uid)
	} else if reflect.DeepEqual(svc.argv, info.Argv) == false {
		t.Fatalf("wrong args: %v", info.Argv)
	} else if svc.NumRecords() != info.Records {
		t.Fatalf("wrong number of records: %d", info.Records)
	} else if svc.NumOracles() != info.Oracles {
		t.Fatalf("wrong number of oracles: %d", info.Oracles)
	}
}

func TestRun(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.Run(nil, &testCall); err != nil {
		t.Fatal(err)
	} else if resp.Success == false {
		t.Fatal("expected success response")
	} else if resp.Msg != "" {
		t.Fatalf("expected empty message: %s", resp.Msg)
	} else if resp.Data == nil {
		t.Fatal("expected response data")
	} else if resp.Data.Compressed == true {
		t.Fatal("expected uncompressed data")
	} else if resp.Data.Payload == nil {
		t.Fatal("expected data payload")
	} else if len(resp.Data.Payload) != 1 || resp.Data.Payload[0] != byte('0') {
		t.Fatalf("unexpected response: %s", resp.Data)
	}
}

func BenchmarkRun(b *testing.B) {
	setup(b, true, true)
	defer teardown(b)

	svc, err := New(testFolder)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		if resp, err := svc.Run(nil, &testCall); err != nil {
			b.Fatal(err)
		} else if resp.Data == nil {
			b.Fatal("expected response data")
		} else if resp.Data.Payload == nil {
			b.Fatal("expected data payload")
		} else if len(resp.Data.Payload) != 1 || resp.Data.Payload[0] != byte('0') {
			b.Fatalf("unexpected response: %s", resp.Data)
		}
	}
}

func TestRunWithCompression(t *testing.T) {
	bak := testOracle.Code
	str := "\"" + strings.Repeat("hello world", 1024) + "\""
	testOracle.Code = "function findReasonsToLive(){ return " + str + "; }"
	defer func() {
		testOracle.Code = bak
	}()

	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.Run(nil, &testCall); err != nil {
		t.Fatal(err)
	} else if resp.Success == false {
		t.Fatalf("expected success response: %v", resp)
	} else if resp.Msg != "" {
		t.Fatalf("expected empty message: %s", resp.Msg)
	} else if resp.Data == nil {
		t.Fatal("expected response data")
	} else if resp.Data.Compressed == false {
		t.Fatal("expected compressed data")
	} else if resp.Data.Payload == nil {
		t.Fatal("expected data payload")
	} else if data := decompress(t, resp.Data); data != str {
		t.Fatalf("unexpected response: %s", data)
	}
}

func BenchmarkRunWithCompression(b *testing.B) {
	bak := testOracle.Code
	str := "\"" + strings.Repeat("hello world", 1024) + "\""
	testOracle.Code = "function findReasonsToLive(){ return " + str + "; }"
	defer func() {
		testOracle.Code = bak
	}()

	setup(b, true, true)
	defer teardown(b)

	svc, err := New(testFolder)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		if resp, err := svc.Run(nil, &testCall); err != nil {
			b.Fatal(err)
		} else if resp.Data == nil {
			b.Fatal("expected response data")
		} else if resp.Data.Compressed == false {
			b.Fatal("expected compressed data")
		} else if resp.Data.Payload == nil {
			b.Fatal("expected data payload")
		}
	}
}

func TestRunWithWithInvalidId(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	call := pb.Call{OracleId: 12345}
	msg := "Oracle 12345 not found."

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.Run(nil, &call); err != nil {
		t.Fatal(err)
	} else if resp == nil {
		t.Fatal("expected error response")
	} else if resp.Success == true {
		t.Fatal("expected error response")
	} else if resp.Msg != msg {
		t.Fatalf("unexpected response message: %s", resp.Msg)
	} else if resp.Data != nil {
		t.Fatalf("unexpected response data: %v", resp.Data)
	}
}

func TestRunWithWithInvalidArgs(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	call := pb.Call{OracleId: 1, Args: []string{"wut,"}}

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.Run(nil, &call); err != nil {
		t.Fatal(err)
	} else if resp == nil {
		t.Fatal("expected error response")
	} else if resp.Success == true {
		t.Fatal("expected error response")
	} else if resp.Data != nil {
		t.Fatalf("unexpected response data: %v", resp.Data)
	}
}

func TestRunWithContextError(t *testing.T) {
	bak := testOracle.Code
	msg := "Error while running oracle 1: nope"
	testOracle.Code = "function findReasonsToLive(){ ctx.Error('nope'); }"
	defer func() {
		testOracle.Code = bak
	}()

	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.Run(nil, &testCall); err != nil {
		t.Fatal(err)
	} else if resp == nil {
		t.Fatal("expected error response")
	} else if resp.Success == true {
		t.Fatal("expected error response")
	} else if resp.Msg != msg {
		t.Fatalf("unexpected response message: %s", resp.Msg)
	} else if resp.Data != nil {
		t.Fatalf("unexpected response data: %v", resp.Data)
	}
}
