package database

import (
	"errors"
	"os"
	"path/filepath"
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

func TestBoltDao_IsEmpty(t *testing.T) {
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

func TestBoltDao_View(t *testing.T) {
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
		t.Error("Expected callback error")
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

func TestBoltDao_Update_CreatesBuckets(t *testing.T) {
	dao, err := OpenBoltDb(testDbFile)
	if err != nil {
		t.Error("Couldn't open test.db")
	} else {
		defer removeTestDb(t, dao)

		dao.Update(func(tx Transaction) error {
			return nil
		})

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

func TestBoltDao_Update_Commits(t *testing.T) {
	dao, err := OpenBoltDb(testDbFile)
	if err != nil {
		t.Error("Couldn't open test.db")
	} else {
		defer removeTestDb(t, dao)

		remoteId := "remoteId"
		err := dao.Update(func(tx Transaction) error {
			if err := tx.InsertFile(remoteId, "text/plain", "name", 16, nil, nil, "1970-01-01", nil); err != nil {
				return err
			}
			return nil
		})

		if err != nil {
			t.Errorf("Unexpected error from Update: %v", err)
		}
		err = dao.db.View(func(tx *bolt.Tx) error {
			file := tx.Bucket([]byte(byIdBucket)).Get([]byte(remoteId))
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

func TestBoltDao_Update_RollsBackOnError(t *testing.T) {
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

func TestBoltDao_FindByPath(t *testing.T) {
	tests := []struct {
		description string
		path        string
		record      *RemoteFile
	}{
		{"file exists", "/remote/file", &RemoteFile{}},
		{"file does not exist", "/unknown/file", nil},
	}

	dao, err := OpenBoltDb(testDbFile)
	if err != nil {
		t.Error("Couldn't open test.db")
	} else {
		defer removeTestDb(t, dao)
		dao.db.Update(func(tx *bolt.Tx) error {
			byPath, _ := tx.CreateBucket([]byte(byPathBucket))
			byPath.Put([]byte(tests[0].path), []byte("fileId"))
			byId, _ := tx.CreateBucket([]byte(byIdBucket))
			byId.Put([]byte("fileId"), toBytes(tests[0].record))
			return nil
		})
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			rf := dao.FindByPath(test.path)
			if rf == nil && test.record != nil {
				t.Fatal("Expected record but got nil")
			}
		})
	}
}