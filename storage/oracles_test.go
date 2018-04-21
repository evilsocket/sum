package storage

import (
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"

	pb "github.com/evilsocket/sum/proto"
	"github.com/golang/protobuf/proto"
)

const (
	testOracles         = 5
	testCorruptedOracle = testFolder + "/666.dat"
)

var (
	testOracle = pb.Oracle{
		Id:   666,
		Name: "findReasonsToLive",
		Code: "function findReasonsToLive(){ return 0; }",
	}
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

func setupOracles(t testing.TB, withValid bool, withCorrupted bool, withBroken bool) {
	log.SetOutput(ioutil.Discard)

	// start clean
	teardownOracles(t)

	if err := os.MkdirAll(testFolder, 0755); err != nil {
		t.Fatalf("Error creating %s: %s", testFolder, err)
	}

	dummy, err := LoadOracles(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	if withValid {
		for i := 1; i <= testOracles; i++ {
			if err := dummy.Create(&testOracle); err != nil {
				t.Fatalf("Error creating oracle: %s", err)
			}
		}
	}

	if withCorrupted {
		if err := ioutil.WriteFile(testCorruptedOracle, []byte("i'm corrupted inside"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	if withBroken {
		if err := dummy.Create(&brokenOracle); err != nil {
			t.Fatalf("Error creating oracle: %s", err)
		}
	}
}

func teardownOracles(t testing.TB) {
	if err := unlink(testFolder); err != nil {
		if !os.IsNotExist(err) {
			t.Fatalf("Error deleting %s: %s", testFolder, err)
		}
	}
}

func TestLoadOracles(t *testing.T) {
	setupOracles(t, true, false, false)
	defer teardownOracles(t)

	oracles, err := LoadOracles(testFolder)
	if err != nil {
		t.Fatal(err)
	} else if oracles == nil {
		t.Fatal("expected valid oracles storage")
	} else if oracles.Size() != testOracles {
		t.Fatalf("expected %d oracles, %d found", testOracles, oracles.Size())
	}

	oracles.ForEach(func(m proto.Message) {
		oracle := m.(*pb.Oracle)
		// id was updated while saving the oracle
		if oracle.Id = testOracle.Id; !reflect.DeepEqual(*oracle, testOracle) {
			t.Fatalf("oracles should be the same here")
		}
	})
}

func TestLoadOraclesWithCorruptedData(t *testing.T) {
	setupOracles(t, true, true, false)
	defer teardownOracles(t)

	if oracles, err := LoadOracles("/lulzlulz"); err == nil {
		t.Fatal("expected error")
	} else if oracles != nil {
		t.Fatal("expected no storage loaded")
	} else if oracles, err := LoadOracles(testFolder); err == nil {
		t.Fatal("expected error due to broken oracle dat file")
	} else if oracles != nil {
		t.Fatal("expected no storage loaded due to corrupted oracle dat file")
	}
}

func TestOraclesFind(t *testing.T) {
	setupOracles(t, true, false, false)
	defer teardownOracles(t)

	oracles, err := LoadOracles(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < testOracles; i++ {
		if compiled := oracles.Find(uint64(i + 1)); compiled == nil {
			t.Fatalf("oracle with id %d not found", i)
		}
	}
}

func TestOraclesFindWithInvalidId(t *testing.T) {
	setupOracles(t, false, false, false)
	defer teardownOracles(t)

	oracles, err := LoadOracles(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < testOracles; i++ {
		if compiled := oracles.Find(uint64(i + 1)); compiled != nil {
			t.Fatalf("oracle with id %d was not expected to be found", i)
		}
	}
}

func TestOraclesUpdate(t *testing.T) {
	setupOracles(t, true, false, false)
	defer teardownOracles(t)

	oracles, err := LoadOracles(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	updatedOracle.Id = 4
	if err := oracles.Update(&updatedOracle); err != nil {
		t.Fatal(err)
	} else if oracle := oracles.Find(updatedOracle.Id); oracle == nil {
		t.Fatalf("expected oracle with id %d", updatedOracle.Id)
	} else if !reflect.DeepEqual(*oracle, updatedOracle) {
		t.Fatal("oracles should match")
	}
}

func TestOraclesDelete(t *testing.T) {
	setupOracles(t, true, false, false)
	defer teardownOracles(t)

	oracles, err := LoadOracles(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < testOracles; i++ {
		id := uint64(i + 1)
		if deleted := oracles.Delete(id); deleted == nil {
			t.Fatalf("oracle with id %d not found", id)
		} else if deleted.Id != id {
			t.Fatalf("should have deleted oracle with id %d, id is %d instead", id, deleted.Id)
		} else if oracles.Size() != uint64(testOracles)-id {
			t.Fatalf("inconsistent oracles storage size of %d", oracles.Size())
		} else if _, err := os.Stat(oracles.pathFor(deleted)); err == nil {
			t.Fatalf("oracle %d data file was not deleted", deleted.Id)
		}
	}

	if oracles.Size() != 0 {
		t.Fatalf("expected empty oracles storage, found %d instead", oracles.Size())
	} else if doublecheck, err := LoadOracles(testFolder); err != nil {
		t.Fatal(err)
	} else if doublecheck.Size() != 0 {
		t.Fatalf("%d dat files left on disk", doublecheck.Size())
	}
}

func TestOraclesDeleteWithInvalidId(t *testing.T) {
	setupOracles(t, false, false, false)
	defer teardownOracles(t)

	oracles, err := LoadOracles(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < testOracles; i++ {
		if deleted := oracles.Delete(uint64(i + 1)); deleted != nil {
			t.Fatalf("oracle with id %d was not expected to be found", i)
		}
	}
}
