package service

import (
	"context"
	"testing"
)

func BenchmarkServiceInfo(b *testing.B) {
	setup(b, true, true)
	defer teardown(b)

	svc, err := New(testFolder, "", "")
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.TODO()
	for i := 0; i < b.N; i++ {
		if _, err := svc.Info(ctx, nil); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkServiceRunWithoutCompression(b *testing.B) {
	setup(b, true, true)
	defer teardown(b)

	svc, err := New(testFolder, "", "")
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.TODO()
	for i := 0; i < b.N; i++ {
		if _, err := svc.Run(ctx, &testCall); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkServiceRunWithCompression(b *testing.B) {
	bak := testOracle.Code

	testOracle.Code = "function findReasonsToLive(){ return " + bigString + "; }"
	defer func() {
		testOracle.Code = bak
	}()

	setup(b, true, true)
	defer teardown(b)

	svc, err := New(testFolder, "", "")
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.TODO()
	for i := 0; i < b.N; i++ {
		if _, err := svc.Run(ctx, &testCall); err != nil {
			b.Fatal(err)
		}
	}
}
