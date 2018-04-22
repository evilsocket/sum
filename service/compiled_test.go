package service

import (
	"testing"
)

func TestServiceCompile(t *testing.T) {
	if compiled, err := compile(&testOracle); err != nil {
		t.Fatal(err)
	} else if compiled == nil {
		t.Fatal("expected compiled oracle")
	} else if compiled.Oracle() != &testOracle {
		t.Fatalf("expected oracle at %p, found %p", &testOracle, compiled.Oracle())
	}
}

func TestServiceCompileWithError(t *testing.T) {
	if compiled, err := compile(&brokenOracle); err == nil {
		t.Fatal("expected compilation error")
	} else if compiled != nil {
		t.Fatal("expected no compiled oracle")
	}
}
