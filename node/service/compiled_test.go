package service

import (
	"errors"
	"github.com/stretchr/testify/require"
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

func TestDontPanic(t *testing.T) {
	var err error
	makef := func(panicArg interface{}) func() {
		return func() {
			defer dontPanic(&err)
			panic(panicArg)
		}
	}

	require.NotPanics(t, makef("string!"))
	require.Error(t, err)
	require.Equal(t, "string!", err.Error())

	require.NotPanics(t, makef(errors.New("error!")))
	require.Error(t, err)
	require.Equal(t, "error!", err.Error())
}
