package service

import (
	"testing"
)

func TestCompile(t *testing.T) {
	if compiled, err := compileOracle(&testOracle); err != nil {
		t.Fatal(err)
	} else if compiled == nil {
		t.Fatal("expected compiled oracle")
	} else if compiled.Oracle() != &testOracle {
		t.Fatalf("expected oracle at %p, found %p", &testOracle, compiled.Oracle())
	}
}

func TestCompileWithError(t *testing.T) {
	if compiled, err := compileOracle(&brokenOracle); err == nil {
		t.Fatal("expected compilation error")
	} else if compiled != nil {
		t.Fatal("expected no compiled oracle")
	}
}

func BenchmarkCompile(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := compileOracle(&testOracle); err != nil {
			b.Fatal(err)
		}
	}
}
