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
		if err := dummy.createUncompiled(&brokenOracle); err != nil {
			t.Fatalf("Error creating oracle: %s", err)
		}
	}
}

func teardownOracles(t testing.TB) {
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
		if o.Id = testOracle.Id; reflect.DeepEqual(*o, testOracle) == false {
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
	setupOracles(b, false, false, false)
	defer teardownOracles(b)

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
	oracles.NextId(1)
	if err := oracles.Create(&testOracle); err == nil {
		t.Fatalf("expected error for non unique oracle id")
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

func BenchmarkOraclesFind(b *testing.B) {
	setupOracles(b, true, false, false)
	defer teardownOracles(b)

	oracles, err := LoadOracles(testFolder)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		id := uint64(i%testOracles) + 1
		if compiled := oracles.Find(id); compiled == nil {
			b.Fatalf("oracle with id %d not found", i)
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

	updatedOracle.Id = 1
	if err := oracles.Update(&updatedOracle); err != nil {
		t.Fatal(err)
	}

	if stored := oracles.Find(updatedOracle.Id); stored == nil {
		t.Fatal("expected stored oracle with id 1")
	} else if reflect.DeepEqual(*stored.Oracle(), updatedOracle) == false {
		t.Fatal("oracle has not been updated as expected")
	}
}

func BenchmarkOraclesUpdate(b *testing.B) {
	setupOracles(b, true, false, false)
	defer teardownOracles(b)

	oracles, err := LoadOracles(testFolder)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		updatedOracle.Id = uint64(i%testOracles) + 1
		if err := oracles.Update(&updatedOracle); err != nil {
			b.Fatal(err)
		}
	}
}

func TestOraclesUpdateInvalidId(t *testing.T) {
	setupOracles(t, true, false, false)
	defer teardownOracles(t)

	oracles, err := LoadOracles(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	updatedOracle.Id = ^uint64(0)
	if err := oracles.Update(&updatedOracle); err == nil {
		t.Fatal("expected error due to invalid id")
	}
}

func TestOraclesUpdateInvalidCode(t *testing.T) {
	setupOracles(t, true, false, false)
	defer teardownOracles(t)

	oracles, err := LoadOracles(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	updatedOracle.Id = 1
	updatedOracle.Code = "lulzlulz"
	if err := oracles.Update(&updatedOracle); err == nil {
		t.Fatal("expected error due to invalid code")
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

func BenchmarkOraclesDelete(b *testing.B) {
	defer teardownOracles(b)

	var oracles *Oracles
	var err error

	for i := 0; i < b.N; i++ {
		// this is not entirely ok as once every 5 times
		// we neeed to recreate and reload oracles, which
		// increases the operations being benchmarked
		id := uint64(i%testOracles) + 1
		if id == 1 {
			setupOracles(b, true, false, false)
			if oracles, err = LoadOracles(testFolder); err != nil {
				b.Fatal(err)
			}
		}

		if deleted := oracles.Delete(id); deleted == nil {
			b.Fatalf("oracle with id %d not found", id)
		}
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
