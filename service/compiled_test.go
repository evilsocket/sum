package service

import (
	"testing"
)

// TODO: make these tests use a list of oracles instead.

func TestServiceCompile(t *testing.T) {
	if compiled, err := compile(&testOracle); err != nil {
		t.Fatal(err)
	} else if compiled == nil {
		t.Fatal("expected compiled oracle")
	} else if !compiled.Is(testOracle) {
		t.Fatalf("expected oracle %v", testOracle)
	}
}

func TestServiceCompileBroken(t *testing.T) {
	if compiled, err := compile(&brokenOracle); err == nil {
		t.Fatal("expected compilation error")
	} else if compiled != nil {
		t.Fatal("expected no compiled oracle")
	} else if err.Error() != "(anonymous): Line 1:6 Unexpected identifier (and 1 more errors)" {
		t.Fatalf("unexpected error message: %s", err)
	}
}

func TestServiceCompileEmpty(t *testing.T) {
	if compiled, err := compile(&emptyOracle); err == nil {
		t.Fatal("expected compilation error")
	} else if compiled != nil {
		t.Fatal("expected no compiled oracle")
	}
}

func TestServiceCompileNoFunctions(t *testing.T) {
	if compiled, err := compile(&noFunctionsOracle); err == nil {
		t.Fatal("expected compilation error")
	} else if compiled != nil {
		t.Fatal("expected no compiled oracle")
	} else if err.Error() != "expected function declaration, found *ast.VariableDeclaration" {
		t.Fatalf("unexpected error message: %s", err)
	}
}

func TestServiceCompileWithErrorDuringDefinition(t *testing.T) {
	if compiled, err := compile(&brokenRunOracle); err == nil {
		t.Fatal("expected compilation error")
	} else if compiled != nil {
		t.Fatal("expected no compiled oracle")
	} else if err.Error() != "ReferenceError: 'not_defined' is not defined" {
		t.Fatalf("unexpected error message: %s", err)
	}
}
