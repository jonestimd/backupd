package database

import (
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

func TestGetFile(t *testing.T) {
	b := makeFileBucket(&remoteFile{Name: "existing", Size: 123})
	tests := []struct {
		description string
		fileId      string
		expected    *remoteFile
	}{
		{"existing file", "existing", toRemoteFile(b.keyValues["existing"])},
		{"non-existing file", "unknown", nil},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			actual := getFile(b, &test.fileId)
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
		expected    []string
	}{
		{"unknown file", makeFileBucket(&remoteFile{Name: "unknown"}), []string{}},
		{"nil parents", makeFileBucket(&remoteFile{Name: "file1"}), []string{"/file1"}},
		{"empty parents", makeFileBucket(&remoteFile{Name: "file1", ParentIds: []string{}}), []string{"/file1"}},
		{"unknown parent", makeFileBucket(&remoteFile{Name: "file1", ParentIds: []string{"parent"}}), []string{"/file1"}},
		{"one parent", makeFileBucket(
			&remoteFile{Name: "file1", ParentIds: []string{"parent"}},
			&remoteFile{Name: "parent", ParentIds: []string{"gp"}}), []string{"/parent/file1"}},
		{"parent with empty parents", makeFileBucket(
			&remoteFile{Name: "file1", ParentIds: []string{"parent"}},
			&remoteFile{Name: "parent", ParentIds: []string{}}), []string{"/parent/file1"}},
		{"two parents", makeFileBucket(
			&remoteFile{Name: "file1", ParentIds: []string{"parent1", "parent2"}},
			&remoteFile{Name: "parent1", ParentIds: []string{"gp"}},
			&remoteFile{Name: "parent2", ParentIds: []string{"gp"}}), []string{"/parent1/file1", "/parent2/file1"}},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			actual := getPaths(test.bucket, "file1")
			if !reflect.DeepEqual(actual, test.expected) {
				t.Errorf("Expected paths %v to equal %v", actual, test.expected)
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
