package service

import (
	"testing"
)

func TestServiceCompiledIs(t *testing.T) {
	if compiled, err := compile(&testOracle); err != nil {
		t.Fatal(err)
	} else if !compiled.Is(testOracle) {
		t.Fatal("compiled object does not match source oracle")
	}
}

func TestServiceCompiledIsNot(t *testing.T) {
	if compiled, err := compile(&testOracle); err != nil {
		t.Fatal(err)
	} else if compiled.Is(brokenOracle) {
		t.Fatal("compiled object should not match a different source oracle")
	}
}
