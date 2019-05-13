package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"github.com/evilsocket/islazy/log"
	"github.com/evilsocket/sum/node/storage"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	pb "github.com/evilsocket/sum/proto"
)

const (
	testFolder  = "/tmp/sum.service.test"
	testRecords = 5
)

var (
	bigString  = "\"" + strings.Repeat("hello world", 1024) + "\""
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

func init() {
	log.Level = log.ERROR
}

func sameRecord(a, b pb.Record) bool {
	return a.Id == b.Id && reflect.DeepEqual(a.Data, b.Data) && reflect.DeepEqual(a.Meta, b.Meta)
}

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
	if d.Compressed {
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
	// start clean
	teardown(t)

	if err := os.MkdirAll(testFolder, 0755); err != nil {
		t.Fatalf("error creating %s: %s", testFolder, err)
	}

	if withRecords {
		basePath := filepath.Join(testFolder, dataFolderName)
		if err := os.MkdirAll(basePath, 0755); err != nil {
			t.Fatalf("error creating folder %s: %s", basePath, err)
		}
		recs, err := storage.LoadRecords(basePath)
		if err != nil {
			t.Fatal(err)
		}
		for i := 1; i <= testRecords; i++ {
			if err := recs.Create(&testRecord); err != nil {
				t.Fatalf("error while creating record: %s", err)
			}
		}
	}

	if withOracles {
		basePath := filepath.Join(testFolder, oraclesFolderName)
		if err := os.MkdirAll(basePath, 0755); err != nil {
			t.Fatalf("error creating folder %s: %s", basePath, err)
		}

		ors, err := storage.LoadOracles(basePath)
		if err != nil {
			t.Fatal(err)
		}
		for i := 1; i <= testOracles; i++ {
			if err := ors.Create(&testOracle); err != nil {
				t.Fatalf("error creating oracle: %s", err)
			}
		}
	}
}

func setupFolders(t testing.TB) {
	// start clean
	teardown(t)

	if err := os.MkdirAll(testFolder, 0755); err != nil {
		t.Fatalf("error creating %s: %s", testFolder, err)
	}

	for _, sub := range []string{dataFolderName, oraclesFolderName} {
		basePath := filepath.Join(testFolder, sub)
		if err := os.MkdirAll(basePath, 0755); err != nil {
			t.Fatalf("error creating folder %s: %s", basePath, err)
		}
	}
}

func teardown(t testing.TB) {
	if err := unlink(testFolder); err != nil {
		if !os.IsNotExist(err) {
			t.Fatalf("error deleting %s: %s", testFolder, err)
		}
	}
}

func TestServiceErrCallResponse(t *testing.T) {
	if r := errCallResponse("test %d", 123); r.Success {
		t.Fatal("success should be false")
	} else if r.Msg != "test 123" {
		t.Fatalf("unexpected message: %s", r.Msg)
	} else if r.Data != nil {
		t.Fatalf("unexpected data pointer: %v", r.Data)
	}
}

func TestServiceNew(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder, "", ""); err != nil {
		t.Fatal(err)
	} else if svc == nil {
		t.Fatal("expected valid service instance")
	} else if time.Since(svc.started).Seconds() >= 1.0 {
		t.Fatalf("wrong started time: %v", svc.started)
	} else if svc.pid != uint64(os.Getpid()) {
		t.Fatalf("wrong pid: %d", svc.pid)
	} else if svc.uid != uint64(os.Getuid()) {
		t.Fatalf("wrong uid: %d", svc.uid)
	} else if !reflect.DeepEqual(svc.argv, os.Args) {
		t.Fatalf("wrong args: %v", svc.argv)
	} else if svc.NumRecords() != testRecords {
		t.Fatalf("wrong number of records: %d", svc.NumRecords())
	} else if svc.NumOracles() != testOracles {
		t.Fatalf("wrong number of oracles: %d", svc.NumOracles())
	}
}

func TestServiceNewWithoutFolders(t *testing.T) {
	defer teardown(t)

	setup(t, false, false)
	if svc, err := New(testFolder, "", ""); err == nil {
		t.Fatal("expected error")
	} else if svc != nil {
		t.Fatal("expected null service instance")
	}

	setup(t, true, false)
	if svc, err := New(testFolder, "", ""); err == nil {
		t.Fatal("expected error")
	} else if svc != nil {
		t.Fatal("expected null service instance")
	}
}

func TestServiceNewWithBrokenCode(t *testing.T) {
	bak := testOracle.Code
	testOracle.Code = "lulz not gonna compile bro"
	defer func() {
		testOracle.Code = bak
	}()

	setup(t, true, true)
	defer teardown(t)

	if _, err := New(testFolder, "", ""); err == nil {
		t.Fatal("expected error due to invalid oracle code")
	}
}

func TestServiceInfo(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder, "", ""); err != nil {
		t.Fatal(err)
	} else if info, err := svc.Info(context.TODO(), nil); err != nil {
		t.Fatal(err)
	} else if info.Version != Version {
		t.Fatalf("wrong version: %s", info.Version)
	} else if info.Uptime > 1 {
		t.Fatalf("wrong uptime: %d", info.Uptime)
	} else if svc.pid != info.Pid {
		t.Fatalf("wrong pid: %d", info.Pid)
	} else if svc.uid != info.Uid {
		t.Fatalf("wrong uid: %d", info.Uid)
	} else if !reflect.DeepEqual(svc.argv, info.Argv) {
		t.Fatalf("wrong args: %v", info.Argv)
	} else if svc.NumRecords() != int(info.Records) {
		t.Fatalf("wrong number of records: %d", info.Records)
	} else if svc.NumOracles() != int(info.Oracles) {
		t.Fatalf("wrong number of oracles: %d", info.Oracles)
	}
}

func TestServiceRun(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder, "", ""); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.Run(context.TODO(), &testCall); err != nil {
		t.Fatal(err)
	} else if !resp.Success {
		t.Fatal("expected success response")
	} else if resp.Msg != "" {
		t.Fatalf("expected empty message: %s", resp.Msg)
	} else if resp.Data == nil {
		t.Fatal("expected response data")
	} else if resp.Data.Compressed {
		t.Fatal("expected uncompressed data")
	} else if resp.Data.Payload == nil {
		t.Fatal("expected data payload")
	} else if len(resp.Data.Payload) != 1 || resp.Data.Payload[0] != byte('0') {
		t.Fatalf("unexpected response: %s", resp.Data)
	}
}

func TestServiceRunWithCompression(t *testing.T) {
	bak := testOracle.Code
	testOracle.Code = "function findReasonsToLive(){ return " + bigString + "; }"
	defer func() {
		testOracle.Code = bak
	}()

	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder, "", ""); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.Run(context.TODO(), &testCall); err != nil {
		t.Fatal(err)
	} else if !resp.Success {
		t.Fatalf("expected success response: %v", resp)
	} else if resp.Msg != "" {
		t.Fatalf("expected empty message: %s", resp.Msg)
	} else if resp.Data == nil {
		t.Fatal("expected response data")
	} else if !resp.Data.Compressed {
		t.Fatal("expected compressed data")
	} else if resp.Data.Payload == nil {
		t.Fatal("expected data payload")
	} else if data := decompress(t, resp.Data); data != bigString {
		t.Fatalf("unexpected response: %s", data)
	}
}

func TestServiceRunWithWithInvalidId(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	call := pb.Call{OracleId: 12345}
	msg := "oracle 12345 not found."

	if svc, err := New(testFolder, "", ""); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.Run(context.TODO(), &call); err != nil {
		t.Fatal(err)
	} else if resp == nil {
		t.Fatal("expected error response")
	} else if resp.Success {
		t.Fatal("expected error response")
	} else if resp.Msg != msg {
		t.Fatalf("unexpected response message: %s", resp.Msg)
	} else if resp.Data != nil {
		t.Fatalf("unexpected response data: %v", resp.Data)
	}
}

func TestServiceRunWithWithInvalidArgs(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	// since the call is precompiled, it doesn't matter
	// what arguments we pass to it
	call := pb.Call{OracleId: 1, Args: []string{"wut,"}}

	if svc, err := New(testFolder, "", ""); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.Run(context.TODO(), &call); err != nil {
		t.Fatal(err)
	} else if resp == nil {
		t.Fatal("expected response")
	} else if !resp.Success {
		t.Fatalf("expected success response, got %v", resp)
	}
}

func TestServiceRunWithWithMissingArgs(t *testing.T) {
	bak := testOracle.Code
	testOracle.Code = `function testMissing(arg){ return (arg || 666); }`
	defer func() {
		testOracle.Code = bak
	}()

	setup(t, true, true)
	defer teardown(t)

	call := pb.Call{OracleId: 1, Args: []string{}}

	if svc, err := New(testFolder, "", ""); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.Run(context.TODO(), &call); err != nil {
		t.Fatal(err)
	} else if resp == nil {
		t.Fatal("expected response")
	} else if !resp.Success {
		t.Fatalf("expected success response, got %v", resp)
	} else if resp.Msg != "" {
		t.Fatalf("expected empty message: %s", resp.Msg)
	} else if resp.Data == nil {
		t.Fatal("expected response data")
	} else if resp.Data.Compressed {
		t.Fatal("expected uncompressed data")
	} else if resp.Data.Payload == nil {
		t.Fatal("expected data payload")
	} else if len(resp.Data.Payload) != 3 || string(resp.Data.Payload) != "666" {
		t.Fatalf("unexpected response: %s", resp.Data)
	}
}

func TestServiceRunWithUnexportableReturn(t *testing.T) {
	bak := testOracle.Code
	msg := "error while running oracle 1: json: unsupported value: +Inf"
	testOracle.Code = "function test(){ return 666 / 0; }"
	defer func() {
		testOracle.Code = bak
	}()
	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder, "", ""); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.Run(context.TODO(), &testCall); err != nil {
		t.Fatal(err)
	} else if resp == nil {
		t.Fatal("expected error response")
	} else if resp.Success {
		t.Fatal("expected error response")
	} else if resp.Msg != msg {
		t.Fatalf("unexpected response message: %s", resp.Msg)
	} else if resp.Data != nil {
		t.Fatalf("unexpected response data: %v", resp.Data)
	}
}

func TestServiceRunWithRuntimeError(t *testing.T) {
	bak := testOracle.Code
	msg := "error while running oracle 1: ReferenceError: 'im_not_defined' is not defined"
	testOracle.Code = "function test(){ return im_not_defined }"
	defer func() {
		testOracle.Code = bak
	}()
	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder, "", ""); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.Run(context.TODO(), &testCall); err != nil {
		t.Fatal(err)
	} else if resp == nil {
		t.Fatal("expected error response")
	} else if resp.Success {
		t.Fatal("expected error response")
	} else if resp.Msg != msg {
		t.Fatalf("unexpected response message: %s", resp.Msg)
	} else if resp.Data != nil {
		t.Fatalf("unexpected response data: %v", resp.Data)
	}
}

func TestServiceRunWithContextError(t *testing.T) {
	bak := testOracle.Code
	msg := "error while running oracle 1: nope"
	testOracle.Code = "function findReasonsToLive(){ ctx.Error('nope'); }"
	defer func() {
		testOracle.Code = bak
	}()

	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder, "", ""); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.Run(context.TODO(), &testCall); err != nil {
		t.Fatal(err)
	} else if resp == nil {
		t.Fatal("expected error response")
	} else if resp.Success {
		t.Fatal("expected error response")
	} else if resp.Msg != msg {
		t.Fatalf("unexpected response message: %s", resp.Msg)
	} else if resp.Data != nil {
		t.Fatalf("unexpected response data: %v", resp.Data)
	}
}
