package storage

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"

	pb "github.com/evilsocket/sum/proto"
)

const (
	testOracles         = 5
	testCorruptedOracle = testFolder + "/666.dat"
)

var (
	updatedOracle = pb.Oracle{
		Id:   666,
		Name: "myNameHasBeenUpdated",
		Code: "function myBodyToo(){ return 0; }",
	}
)

func (o *Oracles) createUncompiled(oracle *pb.Oracle) error {
	o.Lock()
	defer o.Unlock()

	oracle.Id = o.nextId
	o.nextId++

	// make sure the id is unique
	if _, found := o.index[oracle.Id]; found == true {
		return fmt.Errorf("Oracle identifier %d violates the unicity constraint.", oracle.Id)
	}

	o.index[oracle.Id] = nil
	return Flush(oracle, o.pathFor(oracle))
}

func setupOracles(t *testing.T, withValid bool, withCorrupted bool, withBroken bool) {
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
		if err := dummy.createUncompiled(&brokenOracle); err != nil {
			t.Fatalf("Error creating oracle: %s", err)
		}
	}
}

func teardownOracles(t *testing.T) {
	if err := unlink(testFolder); err != nil {
		if os.IsNotExist(err) == false {
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

	oracles.ForEach(func(o *pb.Oracle) {
		// id was updated while saving the oracle
		if reflect.DeepEqual(o, testOracle) == true {
			t.Fatalf("oracles should have different ids")
		} else if o.Id = testOracle.Id; reflect.DeepEqual(*o, testOracle) == false {
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

func TestLoadOraclesWithBrokenCode(t *testing.T) {
	setupOracles(t, true, false, true)
	defer teardownOracles(t)

	if oracles, err := LoadOracles(testFolder); err == nil {
		t.Fatal("expected error due to broken oracle dat file")
	} else if oracles != nil {
		t.Fatal("expected no storage loaded due to broken oracle code")
	}
}

func TestOraclesCreate(t *testing.T) {
	setupOracles(t, false, false, false)
	defer teardownOracles(t)

	oracles, err := LoadOracles(testFolder)
	if err != nil {
		t.Fatal(err)
	} else if oracles.Size() != 0 {
		t.Fatal("expected empty oracle storage")
	} else if err := oracles.Create(&testOracle); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkOraclesCreate(b *testing.B) {
	log.SetOutput(ioutil.Discard)

	oracles, err := LoadOracles(testFolder)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		if err := oracles.Create(&testOracle); err != nil {
			b.Fatal(err)
		}
	}
}

func TestOraclesCreateBroken(t *testing.T) {
	setupOracles(t, false, false, false)
	defer teardownOracles(t)

	oracles, err := LoadOracles(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	if err := oracles.Create(&brokenOracle); err == nil {
		t.Fatalf("expected error due to broken oracle code")
	}
}

func TestOraclesCreateNotUniqueId(t *testing.T) {
	setupOracles(t, true, false, false)
	defer teardownOracles(t)

	oracles, err := LoadOracles(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	// ok this is kinda cheating, but i want full coverage
	oracles.nextId = uint64(1)
	if err := oracles.Create(&testOracle); err == nil {
		t.Fatalf("expected error for non unique oracle id")
	}
}

func TestOraclesUpdate(t *testing.T) {
	setupOracles(t, true, false, false)
	defer teardownOracles(t)

	oracles, err := LoadOracles(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	updatedOracle.Id = 1
	if err := oracles.Update(&updatedOracle); err != nil {
		t.Fatal(err)
	}
}
