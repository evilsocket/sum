package storage

import (
	"testing"
)

func BenchmarkOraclesFind(b *testing.B) {
	setupOracles(b, true, false, false)
	defer teardownOracles(b)

	oracles, err := LoadOracles(testFolder)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		id := uint64(i%testOracles) + 1
		if o := oracles.Find(id); o == nil {
			b.Fatalf("oracle with id %d not found", id)
		}
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
