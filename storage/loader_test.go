package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	pb "github.com/evilsocket/sum/proto"
)

const (
	testFolder  = "/tmp/testsum"
	testBroken  = "/tmp/testsum/bro.ken"
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

func setup(t *testing.T) {
	// start clean
	teardown(t)

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

func teardown(t *testing.T) {
	if err := unlink(testFolder); err != nil {
		if os.IsNotExist(err) == false {
			t.Fatalf("Error deleting %s: %s", testFolder, err)
		}
	}
}

func TestListPath(t *testing.T) {
	setup(t)
	defer teardown(t)

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
		if fileName, found := loadable[idKey]; found == false {
			t.Fatalf("file %s not found", idKey)
		} else if fileName != expected {
			t.Fatalf("expected %s but got %s", fileName, expected)
		}
	}
}

func TestListPathWithError(t *testing.T) {
	if _, _, err := ListPath("/lulzlulz"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestLoad(t *testing.T) {
	setup(t)
	defer teardown(t)

	_, loadable, err := ListPath(testFolder)
	if err != nil {
		t.Fatal(err)
	}

	for _, fileName := range loadable {
		var rec pb.Record
		if err := Load(fileName, &rec); err != nil {
			t.Fatalf("erorr loading %s: %s", fileName, err)
		} else if reflect.DeepEqual(rec, testRecord) == false {
			t.Fatal("records should be the same")
		}
	}
}

func TestLoadWithError(t *testing.T) {
	setup(t)
	defer teardown(t)

	var rec pb.Record

	if err := Load("/lulz.dat", &rec); err == nil {
		t.Fatal("error expected for /lulz.dat")
	} else if err := Load(testBroken, &rec); err == nil {
		t.Fatalf("erorr expected for %s", testBroken)
	}
}
