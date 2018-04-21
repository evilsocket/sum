package service

import (
	"context"
	"reflect"
	"testing"

	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/storage"
)

var (
	testOracles  = 5
	byName       = pb.ByName{Name: "findReasonsToLive"}
	brokenOracle = pb.Oracle{
		Id:   123,
		Name: "brokenOracle",
		Code: "lulz i won't compile =)",
	}
	updatedOracle = pb.Oracle{
		Id:   666,
		Name: "myNameHasBeenUpdated",
		Code: "function myBodyToo(){ return 0; }",
	}
)

func TestServiceErrOracleResponse(t *testing.T) {
	if r := errOracleResponse("test %d", 123); r.Success {
		t.Fatal("success should be false")
	} else if r.Msg != "test 123" {
		t.Fatalf("unexpected message: %s", r.Msg)
	} else if r.Oracles != nil {
		t.Fatalf("unexpected oracles list: %v", r.Oracles)
	}
}

func TestServiceCreateOracle(t *testing.T) {
	setupFolders(t)
	defer teardown(t)

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.CreateOracle(context.TODO(), &testOracle); err != nil {
		t.Fatal(err)
	} else if !resp.Success {
		t.Fatalf("expected success response: %v", resp)
	} else if resp.Oracles != nil {
		t.Fatalf("unexpected oracles list: %v", resp.Oracles)
	} else if resp.Msg != "1" {
		t.Fatalf("unexpected response message: %s", resp.Msg)
	}
}

func TestServiceCreateOracleWithInvalidId(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	svc, err := New(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	svc.oracles.NextID(1)
	if resp, err := svc.CreateOracle(context.TODO(), &testOracle); err != nil {
		t.Fatal(err)
	} else if resp.Success {
		t.Fatalf("expected error response: %v", resp)
	} else if resp.Msg != storage.ErrInvalidID.Error() {
		t.Fatalf("unexpected response message: %s", resp.Msg)
	}
}

func TestServiceCreateBrokenOracle(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.CreateOracle(context.TODO(), &brokenOracle); err != nil {
		t.Fatal(err)
	} else if resp.Success {
		t.Fatalf("expected error response: %v", resp)
	} else if resp.Oracles != nil {
		t.Fatalf("unexpected oracles list: %v", resp.Oracles)
	}
}

func TestServiceUpdateOracle(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	updatedOracle.Id = 1

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.UpdateOracle(context.TODO(), &updatedOracle); err != nil {
		t.Fatal(err)
	} else if !resp.Success {
		t.Fatalf("expected success response: %v", resp)
	} else if resp.Oracles != nil {
		t.Fatalf("unexpected oracles list: %v", resp.Oracles)
	} else if stored := svc.oracles.Find(updatedOracle.Id); stored == nil {
		t.Fatal("expected stored oracle with id 1")
	} else if !reflect.DeepEqual(*stored, updatedOracle) {
		t.Fatal("oracle has not been updated as expected")
	}
}

func TestServiceUpdateOracleWithInvalidId(t *testing.T) {
	setupFolders(t)
	defer teardown(t)

	updatedOracle.Id = 1
	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.UpdateOracle(context.TODO(), &updatedOracle); err != nil {
		t.Fatal(err)
	} else if resp.Success {
		t.Fatalf("expected error response: %v", resp)
	} else if resp.Msg != storage.ErrRecordNotFound.Error() {
		t.Fatalf("unexpected response message: %s", resp.Msg)
	}
}

func TestServiceUpdateWithBrokenOracle(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	brokenOracle.Id = 1

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.UpdateOracle(context.TODO(), &brokenOracle); err != nil {
		t.Fatal(err)
	} else if resp.Success {
		t.Fatalf("expected error response: %v", resp)
	} else if resp.Oracles != nil {
		t.Fatalf("unexpected oracles list: %v", resp.Oracles)
	}
}

func TestServiceReadOracle(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	byID.Id = 1
	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.ReadOracle(context.TODO(), &byID); err != nil {
		t.Fatal(err)
	} else if !resp.Success {
		t.Fatalf("expected success response: %v", resp)
	} else if resp.Oracles == nil {
		t.Fatal("expected oracles list")
	} else if len(resp.Oracles) != 1 {
		t.Fatalf("unexpected oracles list size: %d", len(resp.Oracles))
	} else if testOracle.Id = byID.Id; !reflect.DeepEqual(*resp.Oracles[0], testOracle) {
		t.Fatalf("oracle does not match: %v", resp.Oracles[0])
	}
}

func TestServiceReadOracleWithInvalidId(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.ReadOracle(context.TODO(), &pb.ById{Id: 666}); err != nil {
		t.Fatal(err)
	} else if resp.Success {
		t.Fatalf("expected error response: %v", resp)
	} else if resp.Oracles != nil {
		t.Fatalf("unexpected oracles list: %v", resp.Oracles)
	} else if resp.Msg != "oracle 666 not found." {
		t.Fatalf("unexpected message: %s", resp.Msg)
	}
}

func TestServiceFindOracle(t *testing.T) {
	bak := testOracles
	testOracles = 1
	defer func() {
		testOracles = bak
	}()

	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.FindOracle(context.TODO(), &byName); err != nil {
		t.Fatal(err)
	} else if !resp.Success {
		t.Fatalf("expected success response: %v", resp)
	} else if resp.Oracles == nil {
		t.Fatal("expected oracles list")
	} else if len(resp.Oracles) != testOracles {
		t.Fatalf("unexpected oracles list size: %v", resp.Oracles)
	} else if testOracle.Id = byID.Id; !reflect.DeepEqual(*resp.Oracles[0], testOracle) {
		t.Fatalf("oracle does not match: %v", resp.Oracles[0])
	}
}

func TestServiceFindOracleWithInvalidName(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.FindOracle(context.TODO(), &pb.ByName{Name: "no way i'm an oracle name :D"}); err != nil {
		t.Fatal(err)
	} else if !resp.Success {
		t.Fatalf("expected success response: %v", resp)
	} else if resp.Oracles == nil {
		t.Fatal("expected oracles list")
	} else if len(resp.Oracles) != 0 {
		t.Fatalf("unexpected oracles list size: %v", resp.Oracles)
	}
}

func TestServiceDeleteOracle(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	svc, err := New(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < testOracles; i++ {
		id := uint64(i + 1)
		if resp, err := svc.DeleteOracle(context.TODO(), &pb.ById{Id: id}); err != nil {
			t.Fatal(err)
		} else if !resp.Success {
			t.Fatalf("expected success response: %v", resp)
		} else if resp.Oracles != nil {
			t.Fatalf("unexpected oracles list: %v", resp.Oracles)
		} else if svc.NumOracles() != uint64(testOracles)-id {
			t.Fatalf("inconsistent oracles storage size of %d", svc.NumOracles())
		}
	}

	if svc.NumOracles() != 0 {
		t.Fatalf("expected empty oracles storage, found %d instead", svc.NumOracles())
	} else if doublecheck, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if doublecheck.NumOracles() != 0 {
		t.Fatalf("%d dat files left on disk", doublecheck.NumOracles())
	}
}

func TestServiceDeleteOracleWithInvalidId(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.DeleteOracle(context.TODO(), &pb.ById{Id: 666}); err != nil {
		t.Fatal(err)
	} else if resp.Success {
		t.Fatalf("expected error response: %v", resp)
	} else if resp.Oracles != nil {
		t.Fatalf("unexpected oracles list: %v", resp.Oracles)
	} else if resp.Msg != "Oracle 666 not found." {
		t.Fatalf("unexpected message: %s", resp.Msg)
	}
}
