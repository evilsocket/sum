package master

import (
	"context"
	"testing"
)

func TestServiceCreateDuplicateOracle(t *testing.T) {
	setupFolders(t)
	defer teardown(t)

	if svc, err := NewClient(testFolder); err != nil {
		t.Fatal(err)
	} else if resp, err := svc.CreateOracle(context.TODO(), &testOracle); err != nil {
		t.Fatal(err)
	} else if !resp.Success {
		t.Fatalf("expected success response: %v", resp)
	} else if resp.Oracle != nil {
		t.Fatalf("unexpected oracle: %v", resp.Oracle)
	} else if resp.Msg != "1" {
		t.Fatalf("unexpected response message: %s", resp.Msg)
	} else if resp, err = svc.CreateOracle(context.TODO(), &testOracle); err != nil {
		t.Fatal(err)
	} else if resp.Success {
		t.Fatalf("expected unsuccessful response")
	} else if resp.Msg != "This oracle already exists." {
		t.Fatalf("unexpected response message: %s", resp.Msg)
	}
}
