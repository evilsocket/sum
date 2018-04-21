package service

import (
	"fmt"
	"reflect"
	"testing"

	pb "github.com/evilsocket/sum/proto"
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

func TestErrOracleResponse(t *testing.T) {
	if r := errOracleResponse("test %d", 123); r.Success == true {
		t.Fatal("success should be false")
	} else if r.Msg != "test 123" {
		t.Fatalf("unexpected message: %s", r.Msg)
	} else if r.Oracles != nil {
		t.Fatalf("unexpected oracles list: %v", r.Oracles)
	}
}

func TestCreateOracle(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.CreateOracle(nil, &testOracle); err != nil {
		t.Fatal(err)
	} else if resp.Success == false {
		t.Fatalf("expected success response: %v", resp)
	} else if resp.Oracles != nil {
		t.Fatalf("unexpected oracles list: %v", resp.Oracles)
	} else if resp.Msg != fmt.Sprintf("%d", testOracles+1) {
		t.Fatalf("unexpected response message: %s", resp.Msg)
	}
}

func BenchmarkCreateOracle(b *testing.B) {
	setup(b, true, true)
	defer teardown(b)

	svc, err := New(testFolder)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		if resp, err := svc.CreateOracle(nil, &testOracle); err != nil {
			b.Fatal(err)
		} else if resp.Success == false {
			b.Fatalf("expected success response: %v", resp)
		}
	}
}

func TestCreateBrokenOracle(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.CreateOracle(nil, &brokenOracle); err != nil {
		t.Fatal(err)
	} else if resp.Success == true {
		t.Fatalf("expected error response: %v", resp)
	} else if resp.Oracles != nil {
		t.Fatalf("unexpected oracles list: %v", resp.Oracles)
	}
}

func TestUpdateOracle(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	updatedOracle.Id = 1

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.UpdateOracle(nil, &updatedOracle); err != nil {
		t.Fatal(err)
	} else if resp.Success == false {
		t.Fatalf("expected success response: %v", resp)
	} else if resp.Oracles != nil {
		t.Fatalf("unexpected oracles list: %v", resp.Oracles)
	} else if stored := svc.oracles.Find(updatedOracle.Id); stored == nil {
		t.Fatal("expected stored oracle with id 1")
	} else if reflect.DeepEqual(*stored.Oracle(), updatedOracle) == false {
		t.Fatal("oracle has not been updated as expected")
	}
}

func BenchmarkUpdateOracle(b *testing.B) {
	setup(b, true, true)
	defer teardown(b)

	updatedOracle.Id = 1
	svc, err := New(testFolder)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		if resp, err := svc.UpdateOracle(nil, &updatedOracle); err != nil {
			b.Fatal(err)
		} else if resp.Success == false {
			b.Fatalf("expected success response: %v", resp)
		}
	}
}

func TestUpdateWithBrokenOracle(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	brokenOracle.Id = 1

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.UpdateOracle(nil, &brokenOracle); err != nil {
		t.Fatal(err)
	} else if resp.Success == true {
		t.Fatalf("expected error response: %v", resp)
	} else if resp.Oracles != nil {
		t.Fatalf("unexpected oracles list: %v", resp.Oracles)
	}
}

func TestReadOracle(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	byId.Id = 1
	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.ReadOracle(nil, &byId); err != nil {
		t.Fatal(err)
	} else if resp.Success == false {
		t.Fatalf("expected success response: %v", resp)
	} else if resp.Oracles == nil {
		t.Fatal("expected oracles list")
	} else if len(resp.Oracles) != 1 {
		t.Fatalf("unexpected oracles list size: %d", len(resp.Oracles))
	} else if testOracle.Id = byId.Id; reflect.DeepEqual(*resp.Oracles[0], testOracle) == false {
		t.Fatalf("oracle does not match: %v", resp.Oracles[0])
	}
}

func BenchmarkReadOracle(b *testing.B) {
	setup(b, true, true)
	defer teardown(b)

	svc, err := New(testFolder)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		byId.Id = uint64(i%testOracles) + 1
		if resp, err := svc.ReadOracle(nil, &byId); err != nil {
			b.Fatal(err)
		} else if resp.Success == false {
			b.Fatalf("expected success response: %v", resp)
		}
	}
}

func TestReadOracleWithInvalidId(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.ReadOracle(nil, &pb.ById{Id: 666}); err != nil {
		t.Fatal(err)
	} else if resp.Success == true {
		t.Fatalf("expected error response: %v", resp)
	} else if resp.Oracles != nil {
		t.Fatalf("unexpected oracles list: %v", resp.Oracles)
	} else if resp.Msg != "Oracle 666 not found." {
		t.Fatalf("unexpected message: %s", resp.Msg)
	}
}

func TestFindOracle(t *testing.T) {
	bak := testOracles
	testOracles = 1
	defer func() {
		testOracles = bak
	}()

	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.FindOracle(nil, &byName); err != nil {
		t.Fatal(err)
	} else if resp.Success == false {
		t.Fatalf("expected success response: %v", resp)
	} else if resp.Oracles == nil {
		t.Fatal("expected oracles list")
	} else if len(resp.Oracles) != testOracles {
		t.Fatalf("unexpected oracles list size: %v", resp.Oracles)
	} else if testOracle.Id = byId.Id; reflect.DeepEqual(*resp.Oracles[0], testOracle) == false {
		t.Fatalf("oracle does not match: %v", resp.Oracles[0])
	}
}

func BenchmarkFindOracle(b *testing.B) {
	bak := testOracles
	testOracles = 1
	defer func() {
		testOracles = bak
	}()

	setup(b, true, true)
	defer teardown(b)

	svc, err := New(testFolder)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		if resp, err := svc.FindOracle(nil, &byName); err != nil {
			b.Fatal(err)
		} else if resp.Success == false {
			b.Fatalf("expected success response: %v", resp)
		}
	}
}

func TestFindOracleWithInvalidName(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.FindOracle(nil, &pb.ByName{Name: "no way i'm an oracle name :D"}); err != nil {
		t.Fatal(err)
	} else if resp.Success == false {
		t.Fatalf("expected success response: %v", resp)
	} else if resp.Oracles == nil {
		t.Fatal("expected oracles list")
	} else if len(resp.Oracles) != 0 {
		t.Fatalf("unexpected oracles list size: %v", resp.Oracles)
	}
}

func TestDeleteOracle(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	svc, err := New(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < testOracles; i++ {
		id := uint64(i + 1)
		if resp, err := svc.DeleteOracle(nil, &pb.ById{Id: id}); err != nil {
			t.Fatal(err)
		} else if resp.Success == false {
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

func BenchmarkDeleteOracle(b *testing.B) {
	var svc *Service
	var err error

	defer teardown(b)
	for i := 0; i < b.N; i++ {
		// this is not entirely ok as once every 5 times
		// we neeed to recreate and reload oracles, which
		// increases the operations being benchmarked
		id := uint64(i%testOracles) + 1
		if id == 1 {
			setup(b, true, true)
			if svc, err = New(testFolder); err != nil {
				b.Fatal(err)
			}
		}

		if resp, err := svc.DeleteOracle(nil, &pb.ById{Id: id}); err != nil {
			b.Fatal(err)
		} else if resp.Success == false {
			b.Fatalf("expected success response: %v", resp)
		}
	}
}

func TestDeleteOracleWithInvalidId(t *testing.T) {
	setup(t, true, true)
	defer teardown(t)

	if svc, err := New(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.DeleteOracle(nil, &pb.ById{Id: 666}); err != nil {
		t.Fatal(err)
	} else if resp.Success == true {
		t.Fatalf("expected error response: %v", resp)
	} else if resp.Oracles != nil {
		t.Fatalf("unexpected oracles list: %v", resp.Oracles)
	} else if resp.Msg != "Oracle 666 not found." {
		t.Fatalf("unexpected message: %s", resp.Msg)
	}
}
