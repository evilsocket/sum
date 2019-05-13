package service

import (
	"bytes"
	"github.com/evilsocket/sum/node/storage"
	"math/rand"
	"testing"

	pb "github.com/evilsocket/sum/proto"
)

const (
	dummyFunc = `function dummy(){}`

	simpleAdd = `function add(a, b){ return a + b; }`

	fiboIter = `function fibonacci(num){
	  var a = 1, b = 0, temp;
	  while (num >= 0){
		temp = a;
		a = a + b;
		b = temp;
		num--;
	  }
	  return b;
	}`

	fiboRecu = `function fibonacci(num) {
	  if (num <= 1) return 1;

	  return fibonacci(num - 1) + fibonacci(num - 2);
	}`

	fiboMemo = `function fibonacci(num, memo) {
	  memo = memo || {};

	  if (memo[num]) return memo[num];
	  if (num <= 1) return 1;

	  return memo[num] = fibonacci(num - 1, memo) + fibonacci(num - 2, memo);
	}`

	findSimilar = `function findSimilar(id, threshold) {
		var v = records.Find(id);
		if( v.IsNull() == true ) {
			return ctx.Error("Vector " + id + " not found.");
		}

		var results = {};
		var all = records.AllBut(v)
		var num = all.length;    
		for( var i = 0; i < num; ++i ) {
			var record = all[i];
			var similarity = v.Cosine(record);
			if( similarity >= threshold ) {
			   results[record.Id] = similarity
			}
		}

		return results;
	}`
)

func BenchmarkCompile(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := compile(&testOracle); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompiledIs(b *testing.B) {
	oracle := pb.Oracle{Name: "simple", Code: "function simple(){ return 0; }"}
	compiled, err := compile(&oracle)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !compiled.Is(oracle) {
			b.Fatal("compiled object does not match source oracle")
		}
	}
}

func benchVM(b *testing.B, fname, code string, args []string, expected string, records *storage.Records) {
	oracle := pb.Oracle{
		Name: fname,
		Code: code,
	}
	compiled, err := compile(&oracle)
	if err != nil {
		b.Fatal(err)
	}

	exp := []byte(nil)
	if expected != "" {
		exp = []byte(expected)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if ctx, ret, err := compiled.Run(records, args); err != nil {
			b.Fatal(err)
		} else if ctx.IsError() {
			b.Fatal(ctx.Message())
		} else if exp != nil && !bytes.Equal(ret, exp) {
			b.Fatalf("expected '%s', got '%s'", expected, ret)
		}
	}
}

func BenchmarkRunDummy(b *testing.B) {
	benchVM(b, "dummy", dummyFunc, nil, "", nil)
}

func BenchmarkRunAdd(b *testing.B) {
	benchVM(b, "add", simpleAdd, []string{"12", "34"}, "46", nil)
}

func BenchmarkRunIterativeFibonacci(b *testing.B) {
	benchVM(b, "fibonacci", fiboIter, []string{"25"}, "121393", nil)
}

func BenchmarkRunRecursiveFibonacci(b *testing.B) {
	benchVM(b, "fibonacci", fiboRecu, []string{"25"}, "121393", nil)
}

func BenchmarkRunMemoFibonacci(b *testing.B) {
	benchVM(b, "fibonacci", fiboMemo, []string{"25"}, "121393", nil)
}

func runFindSimilar(b *testing.B, rows int, cols int) {
	setupFolders(b)
	defer teardown(b)

	records, err := storage.LoadRecords(testFolder)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < rows; i++ {
		record := pb.Record{
			Data: make([]float32, cols),
		}
		for j := 0; j < cols; j++ {
			record.Data[j] = rand.Float32()
		}

		if err := records.Create(&record); err != nil {
			b.Fatal(err)
		}
	}

	benchVM(b, "findSimilar", findSimilar, []string{"1", "0.9"}, "", records)
}

func BenchmarkRunFindSimilar10x100(b *testing.B) {
	runFindSimilar(b, 10, 100)
}

func BenchmarkRunFindSimilar10x500(b *testing.B) {
	runFindSimilar(b, 10, 500)
}

func BenchmarkRunFindSimilar10x1000(b *testing.B) {
	runFindSimilar(b, 10, 1000)
}

func BenchmarkRunFindSimilar100x10(b *testing.B) {
	runFindSimilar(b, 100, 10)
}

func BenchmarkRunFindSimilar200x10(b *testing.B) {
	runFindSimilar(b, 200, 10)
}

func BenchmarkRunFindSimilar100x1(b *testing.B) {
	runFindSimilar(b, 100, 1)
}

func BenchmarkRunFindSimilar10000x50(b *testing.B) {
	runFindSimilar(b, 10000, 50)
}
