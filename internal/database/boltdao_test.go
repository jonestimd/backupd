package database

import (
	"bytes"
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	bolt "github.com/coreos/bbolt"
)

const (
	dataDir = "testdata"
)

var testDbFile string
var emptyDbFile string
var nonemptyDbFile string

func init() {
	testDbFile = filepath.Join(dataDir, "test.db")
	emptyDbFile = filepath.Join(dataDir, "empty.db")
	nonemptyDbFile = filepath.Join(dataDir, "non-empty.db")
}

type mockBucket struct {
	files map[string]*remoteFile
}

func (b *mockBucket) Get(key []byte) []byte {
	file := b.files[string(key)]
	if file != nil {
		buf := bytes.Buffer{}
		enc := gob.NewEncoder(&buf)
		enc.Encode(file)
		return buf.Bytes()
	}
	return nil
}

func makeMockBucket(files ...*remoteFile) *mockBucket {
	b := &mockBucket{make(map[string]*remoteFile)}
	for _, file := range files {
		b.files[file.Name] = file
	}
	return b
}

func TestGetFile(t *testing.T) {
	b := makeMockBucket(&remoteFile{Name: "existing", Size: 123})
	tests := []struct {
		description string
		fileId      string
		expected    *remoteFile
	}{
		{"existing file", "existing", b.files["existing"]},
		{"non-existing file", "unknown", nil},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			actual := getFile(b, test.fileId)
			if !reflect.DeepEqual(actual, test.expected) {
				t.Errorf("Expected remoteFile %v to equal %v", actual, test.expected)
			}
		})
	}
}

func TestGetPath(t *testing.T) {
	tests := []struct {
		description string
		bucket      *mockBucket
		expected    string
	}{
		{"nil parents", makeMockBucket(&remoteFile{Name: "name"}), "/"},
		{"no parents", makeMockBucket(&remoteFile{Name: "file1", ParentIds: []string{"parent"}}), "/file1"},
		{"one parent", makeMockBucket(
			&remoteFile{Name: "file1", ParentIds: []string{"parent"}},
			&remoteFile{Name: "parent", ParentIds: []string{"gp"}}), "/parent/file1"},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			actual := getPath(test.bucket, "file1")
			if actual != test.expected {
				t.Errorf("Expected path %s to equal %s", actual, test.expected)
			}
		})
	}
}

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		description string
		dbFile      string
		expected    bool
	}{
		{"new database", emptyDbFile, true},
		{"database with bucket", nonemptyDbFile, false},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			dao, err := OpenBoltDb(test.dbFile)
			if err != nil {
				t.Errorf("failed to open database %s", test.dbFile)
			} else {
				defer dao.Close()
				if dao.IsEmpty() != test.expected {
					t.Errorf("Expected IsEmpty() = %v for %s", test.expected, test.dbFile)
				}
			}
		})
	}
}

func TestView(t *testing.T) {
	dao, err := OpenBoltDb(nonemptyDbFile)
	if err != nil {
		t.Errorf("Unexpected error from open: %v", err)
	}
	defer dao.Close()
	cbError := errors.New("callback error")

	err = dao.View(func(tx Transaction) error {
		return cbError
	})

	if err != cbError {
		t.Errorf("Expected callback error")
	}
}

func removeTestDb(t *testing.T, dao Dao) {
	if err := dao.Close(); err != nil {
		t.Logf("Error closing test.db: %v", err)
	}
	if err := os.Remove(testDbFile); err != nil && !os.IsNotExist(err) {
		t.Logf("Error deleting test.db: %v", err)
	}
}

func TestUpdateCreatesBuckets(t *testing.T) {
	dao, err := OpenBoltDb(testDbFile)
	if err != nil {
		t.Error("Couldn't open test.db")
	} else {
		defer removeTestDb(t, dao)

		dao.Update(func(tx Transaction) error { return nil })

		dao.db.View(func(tx *bolt.Tx) error {
			if tx.Bucket([]byte(byIdBucket)) == nil {
				t.Errorf("%s bucket not created", byIdBucket)
			}
			if tx.Bucket([]byte(byPathBucket)) == nil {
				t.Errorf("%s bucket not created", byPathBucket)
			}
			return nil
		})
	}
}

func TestUpdateCommits(t *testing.T) {
	dao, err := OpenBoltDb(testDbFile)
	if err != nil {
		t.Error("Couldn't open test.db")
	} else {
		defer removeTestDb(t, dao)

		err := dao.Update(func(tx Transaction) error {
			if err := tx.InsertFile("fileId", "name", 16, nil, nil, "1970-01-01", nil); err != nil {
				return err
			}
			return nil
		})

		if err != nil {
			t.Errorf("Unexpected error from Update: %v", err)
		}
		err = dao.db.View(func(tx *bolt.Tx) error {
			file := tx.Bucket([]byte(byIdBucket)).Get([]byte("fileId"))
			if file == nil {
				t.Error("Expected file to be saved")
			}
			return nil
		})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestUpdateRollsbackOnError(t *testing.T) {
	dao, err := OpenBoltDb(testDbFile)
	if err != nil {
		t.Error("Couldn't open test.db")
	} else {
		defer removeTestDb(t, dao)
		cbError := errors.New("callback failure")

		err := dao.Update(func(tx Transaction) error {
			return cbError
		})

		if err != cbError {
			t.Error("expected callback error")
		}
		err = dao.db.View(func(tx *bolt.Tx) error {
			if tx.Bucket([]byte(byIdBucket)) != nil {
				t.Error("Expected create bucket to be rolled back")
			}
			return nil
		})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}
