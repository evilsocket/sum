package storage

import (
	"testing"

	pb "github.com/evilsocket/sum/proto"
)

var (
	testOracle = pb.Oracle{
		Id:   666,
		Name: "findReasonsToLive",
		Code: "function findReasonsToLive(){ return 0; }",
	}
	brokenOracle = pb.Oracle{
		Id:   123,
		Name: "brokenOracle",
		Code: "lulz i won't compile =)",
	}
)

func TestCompile(t *testing.T) {
	if compiled, err := Compile(&testOracle); err != nil {
		t.Fatal(err)
	} else if compiled == nil {
		t.Fatal("expected compiled oracle")
	} else if compiled.Oracle() != &testOracle {
		t.Fatalf("expected oracle at %p, found %p", &testOracle, compiled.Oracle())
	} else if compiled.VM() == nil {
		t.Fatal("expected valid vm")
	}
}

func TestCompileWithError(t *testing.T) {
	if compiled, err := Compile(&brokenOracle); err == nil {
		t.Fatal("expected compilation error")
	} else if compiled != nil {
		t.Fatal("expected no compiled oracle")
	}
}

func BenchmarkCompile(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := Compile(&testOracle); err != nil {
			b.Fatal(err)
		}
	}
}

func TestRun(t *testing.T) {
	if compiled, err := Compile(&testOracle); err != nil {
		t.Fatal(err)
	} else if _, err := compiled.Run([]string{}); err != nil {
		t.Fatal(err)
	}
}

func TestRunWithError(t *testing.T) {
	if compiled, err := Compile(&testOracle); err != nil {
		t.Fatal(err)
	} else if _, err := compiled.Run([]string{"im_not_defined"}); err == nil {
		t.Fatal("expected error")
	}
}

func BenchmarkRun(b *testing.B) {
	compiled, err := Compile(&testOracle)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		if _, err := compiled.Run([]string{}); err != nil {
			b.Fatal(err)
		}
	}
}
