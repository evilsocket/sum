package service

import (
	"bytes"
	"testing"

	pb "github.com/evilsocket/sum/proto"
)

const (
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
)

func BenchmarkCompileOracle(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := compileOracle(&testOracle); err != nil {
			b.Fatal(err)
		}
	}
}

func benchVM(b *testing.B, fname, code string, args []string, expected string) {
	oracle := pb.Oracle{
		Name: fname,
		Code: code,
	}
	compiled, err := compileOracle(&oracle)
	if err != nil {
		b.Fatal(err)
	}

	exp := []byte(nil)
	if expected != "" {
		exp = []byte(expected)
	}

	for i := 0; i < b.N; i++ {
		if ctx, ret, err := compiled.RunWithContext(nil, args); err != nil {
			b.Fatal(err)
		} else if ctx.IsError() {
			b.Fatal(ctx.Message())
		} else if exp != nil && !bytes.Equal(ret, exp) {
			b.Fatalf("expected '%s', got '%s'", expected, ret)
		}
	}
}

func BenchmarkRunDummyWithContext(b *testing.B) {
	benchVM(b, "dummy", "function dummy(){}", nil, "")
}

func BenchmarkRunAddWithContext(b *testing.B) {
	benchVM(b, "add", "function add(a, b){ return a + b; }", []string{"12", "34"}, "46")
}

func BenchmarkRunIterativeFibonacciWithContext(b *testing.B) {
	benchVM(b, "fibonacci", fiboIter, []string{"25"}, "121393")
}

func BenchmarkRunRecursiveFibonacciWithContext(b *testing.B) {
	benchVM(b, "fibonacci", fiboRecu, []string{"25"}, "121393")
}

func BenchmarkRunMemoFibonacciWithContext(b *testing.B) {
	benchVM(b, "fibonacci", fiboMemo, []string{"25"}, "121393")
}
