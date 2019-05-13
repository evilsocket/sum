package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	pb "github.com/evilsocket/sum/proto"
)

const (
	testFolder  = "/tmp/sum.storage.test"
	testBroken  = testFolder + "/bro.ken"
	testRecords = 5
)

func unlink(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func setupRawRecords(t testing.TB) {
	// start clean
	teardownRecords(t)

	if err := os.MkdirAll(testFolder, 0755); err != nil {
		t.Fatalf("Error creating %s: %s", testFolder, err)
	}

	for i := 1; i <= testRecords; i++ {
		fileName := filepath.Join(testFolder, fmt.Sprintf("%d.dat", i))
		if err := Flush(&testRecord, fileName); err != nil {
			t.Fatalf("Error writing to %s: %s", testFolder, err)
		}
	}

	if err := ioutil.WriteFile(testBroken, []byte("i'm broken inside"), 0755); err != nil {
		t.Fatal(err)
	}
}

func teardownRecords(t testing.TB) {
	if err := unlink(testFolder); err != nil {
		if !os.IsNotExist(err) {
			t.Fatalf("Error deleting %s: %s", testFolder, err)
		}
	}
}

// ugly but better than writing platform specific implementations
func canWriteOnRoot() bool {
	testFile := "/root/.sum.w.test"
	if err := ioutil.WriteFile(testFile, []byte{0x00}, 0755); err == nil {
		os.Remove(testFile)
		return true
	}
	return false
}

func TestLoaderListPath(t *testing.T) {
	setupRawRecords(t)
	defer teardownRecords(t)

	path, loadable, err := ListPath(testFolder)
	if err != nil {
		t.Fatal(err)
	} else if path != testFolder {
		t.Fatalf("path (%s) should be '%s'", path, testFolder)
	} else if len(loadable) != testRecords {
		t.Fatalf("expected %d files, got %d", testRecords, len(loadable))
	}

	for i := 1; i <= testRecords; i++ {
		idKey := fmt.Sprintf("%d", i)
		expected := filepath.Join(testFolder, fmt.Sprintf("%d.dat", i))
		if fileName, found := loadable[idKey]; !found {
			t.Fatalf("file %s not found", idKey)
		} else if fileName != expected {
			t.Fatalf("expected %s but got %s", fileName, expected)
		}
	}
}

func TestLoaderListPathWithError(t *testing.T) {
	if _, _, err := ListPath("/dev/random"); err == nil {
		t.Fatal("expected an error")
	} else if _, _, err := ListPath("/lulzlulz"); err == nil {
		t.Fatal("expected an error")
	} else if !canWriteOnRoot() {
		// on docker this check is skipped
		if _, _, err := ListPath("/root"); err == nil {
			t.Fatal("expected permission denied")
		}
	}
}

func TestLoaderLoad(t *testing.T) {
	setupRawRecords(t)
	defer teardownRecords(t)

	_, loadable, err := ListPath(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	for _, fileName := range loadable {
		var rec pb.Record
		if err := Load(fileName, &rec); err != nil {
			t.Fatalf("erorr loading %s: %s", fileName, err)
		} else if !sameRecord(rec, testRecord) {
			t.Fatal("records should be the same")
		}
	}
}

func TestLoaderLoadWithError(t *testing.T) {
	setupRawRecords(t)
	defer teardownRecords(t)

	var rec pb.Record

	if err := Load("/lulz.dat", &rec); err == nil {
		t.Fatal("error expected for /lulz.dat")
	} else if err := Load(testBroken, &rec); err == nil {
		t.Fatalf("erorr expected for %s", testBroken)
	}
}
