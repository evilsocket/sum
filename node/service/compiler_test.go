package service

import (
	"strings"
	"testing"

	pb "github.com/evilsocket/sum/proto"
)

var units = []struct {
	oracle               pb.Oracle
	expectedError        bool
	expectedErrorMessage string
}{
	{pb.Oracle{Name: "empty"}, true, "expected a function declaration"},
	{pb.Oracle{Name: "simple", Code: "function simple(){ return 0; }"}, false, ""},
	{pb.Oracle{Name: "broken", Code: "lulz i won't compile =)"}, true, "unexpected identifier"},
	{pb.Oracle{Name: "no functions", Code: "var lulz = 123;"}, true, "expected a function declaration"},
	{pb.Oracle{Name: "error during definition", Code: "function imok(){} imnot = not_defined + 1;"}, true, "ReferenceError"},
}

func TestServiceCompiler(t *testing.T) {
	for _, u := range units {
		t.Run(u.oracle.Name, func(t *testing.T) {
			compiled, err := compile(&u.oracle)
			if u.expectedError {
				if err == nil {
					t.Fatal("an error was expected")
				} else if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(u.expectedErrorMessage)) {
					t.Fatalf("expected error message '%s', got '%s'", u.expectedErrorMessage, err)
				}
			} else if err != nil {
				t.Fatal(err)
			} else if !compiled.Is(u.oracle) {
				t.Fatal("compiled oracle does not match source object")
			}
		})
	}
}
