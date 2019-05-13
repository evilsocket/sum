package wrapper

import (
	"testing"
)

var (
	testErrorMessage = "foobar"
)

func TestWrapperContext(t *testing.T) {
	ctx := NewContext()
	if ctx == nil {
		t.Fatal("expectd valid context")
	} else if ctx.IsError() {
		t.Fatal("expected neutral state")
	} else if ctx.Message() != "" {
		t.Fatal("expected empty message")
	} else if ctx.Error(testErrorMessage); !ctx.IsError() {
		t.Fatal("expected error state")
	} else if ctx.Message() != testErrorMessage {
		t.Fatalf("expected '%s', got '%s'", testErrorMessage, ctx.Message())
	} else if ctx.Reset(); ctx.IsError() {
		t.Fatal("expected neutral state again")
	} else if ctx.Message() != "" {
		t.Fatal("expected empty message again")
	}
}
