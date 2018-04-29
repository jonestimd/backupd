package database

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	bolt "github.com/coreos/bbolt"
	"github.com/jonestimd/backupd/internal/filesys"
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

func TestBoltDao_OpenDb_existingFile(t *testing.T) {
	getFiles := func() (chan FileOrError, error) {
		return nil, errors.New("Unexpected call to getFiles")
	}

	dao, err := OpenDb(nonemptyDbFile, getFiles)

	if err != nil {
		t.Errorf("failed to open database %s: %s", nonemptyDbFile, err.Error())
	} else {
		defer dao.Close()
		dao.db.View(func(tx *bolt.Tx) error {
			byID := tx.Bucket([]byte(byIDBucket))
			if byID.Stats().KeyN != 0 {
				t.Errorf("Expected no file recordss, got %d", byID.Stats().KeyN)
			}
			byPath := tx.Bucket([]byte(byPathBucket))
			if byPath.Stats().KeyN != 0 {
				t.Errorf("Expected no path recordss, got %d", byPath.Stats().KeyN)
			}
			return nil
		})
	}
}

func TestBoltDao_OpenDb_invalidFile(t *testing.T) {
	dao, err := OpenDb(filepath.FromSlash("/test.db"), nil)

	if err == nil {
		dao.Close()
		t.Error("Expected an error opening /test.db")
	}
}

func checkEmptyDb(t *testing.T) {
	db, err := bolt.Open(emptyDbFile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		t.Errorf("Error opening file %s: %s", emptyDbFile, err.Error())
	} else {
		defer db.Close()
		db.View(func(tx *bolt.Tx) error {
			byID := tx.Bucket([]byte(byIDBucket))
			if byID != nil {
				t.Errorf("Bucket %s exists, expected transaction to be rolled back", byIDBucket)
			}
			byPath := tx.Bucket([]byte(byPathBucket))
			if byPath != nil {
				t.Errorf("Bucket %s exists, expected transaction to be rolled back", byPathBucket)
			}
			return nil
		})
	}
}

func TestBoltDao_OpenDb_initializeEmptyDatabase(t *testing.T) {
	getFiles := func() (chan FileOrError, error) {
		ch := make(chan FileOrError, 0)
		go func() {
			ch <- FileOrError{File: NewRemoteFile("name", "plain/text", 10, "checksum", []string{}, "2011-01-01", "local ID", "remote ID")}
			ch <- FileOrError{Error: errors.New("Rollback initialize")}
			close(ch)
		}()
		return ch, nil
	}

	dao, err := OpenDb(emptyDbFile, getFiles)

	if err == nil {
		dao.Close()
		t.Error("Expected the error from getFiles()")
	} else if err.Error() != "Rollback initialize" {
		t.Errorf("Unexpected error: %s", err.Error())
	} else {
		checkEmptyDb(t)
	}
}

func TestBoltDao_OpenDb_errorLoadingFiles(t *testing.T) {
	getFiles := func() (chan FileOrError, error) {
		return nil, errors.New("error laoding files")
	}

	dao, err := OpenDb(emptyDbFile, getFiles)

	if err == nil {
		dao.Close()
		t.Error("Expected the error from getFiles()")
	} else if err.Error() != "error laoding files" {
		t.Errorf("Unexpected error: %s", err.Error())
	} else {
		checkEmptyDb(t)
	}
}

func TestBoltDao_OpenDb_createFile(t *testing.T) {
	getFiles := func() (chan FileOrError, error) {
		ch := make(chan FileOrError, 0)
		go func() {
			ch <- FileOrError{File: NewRemoteFile("name", "plain/text", 10, "checksum", []string{}, "2011-01-01", "local ID", "remote ID")}
			close(ch)
		}()
		return ch, nil
	}
	defer removeTestDb(t, nil)

	dao, err := OpenDb(testDbFile, getFiles)

	if err != nil {
		t.Errorf("failed to open database %s: %s", testDbFile, err.Error())
	} else {
		defer dao.Close()
		dao.db.View(func(tx *bolt.Tx) error {
			byID := tx.Bucket([]byte(byIDBucket))
			if byID == nil {
				t.Errorf("%s bucket not created", byIDBucket)
			}
			if byID.Stats().KeyN != 1 {
				t.Errorf("Expected 1 file record, got %d", byID.Stats().KeyN)
			}
			byPath := tx.Bucket([]byte(byPathBucket))
			if byPath == nil {
				t.Errorf("%s bucket not created", byPathBucket)
			}
			if byPath.Stats().KeyN != 1 {
				t.Errorf("Expected 1 path record, got %d", byPath.Stats().KeyN)
			}
			return nil
		})
	}
}

func TestBoltDao_isEmpty(t *testing.T) {
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
			dao, err := OpenDb(test.dbFile, nil)
			if err != nil {
				t.Errorf("failed to open database %s: %s", test.dbFile, err.Error())
			} else {
				defer dao.Close()
				if dao.isEmpty() != test.expected {
					t.Errorf("Expected IsEmpty() = %v for %s", test.expected, test.dbFile)
				}
			}
		})
	}
}

func removeTestDb(t *testing.T, dao *BoltDao) {
	if dao != nil {
		if err := dao.Close(); err != nil {
			t.Logf("Error closing test.db: %v", err)
		}
	}
	if err := os.Remove(testDbFile); err != nil && !os.IsNotExist(err) {
		t.Logf("Error deleting test.db: %v", err)
	}
}

func TestBoltDao_Update_CreatesBuckets(t *testing.T) {
	dao, err := OpenDb(testDbFile, nil)
	if err != nil {
		t.Error("Couldn't open test.db")
	} else {
		defer removeTestDb(t, dao)

		dao.update(func(tx *boltTx) error {
			return nil
		})

		dao.db.View(func(tx *bolt.Tx) error {
			if tx.Bucket([]byte(byIDBucket)) == nil {
				t.Errorf("%s bucket not created", byIDBucket)
			}
			if tx.Bucket([]byte(byPathBucket)) == nil {
				t.Errorf("%s bucket not created", byPathBucket)
			}
			return nil
		})
	}
}

func TestBoltDao_Update_Commits(t *testing.T) {
	dao, err := OpenDb(testDbFile, nil)
	if err != nil {
		t.Error("Couldn't open test.db")
	} else {
		defer removeTestDb(t, dao)

		remoteID := "remoteId"
		err := dao.update(func(tx *boltTx) error {
			if err := tx.insertFile(remoteID, "text/plain", "name", 16, nil, nil, "1970-01-01", nil); err != nil {
				return err
			}
			return nil
		})

		if err != nil {
			t.Errorf("Unexpected error from Update: %v", err)
		}
		err = dao.db.View(func(tx *bolt.Tx) error {
			file := tx.Bucket([]byte(byIDBucket)).Get([]byte(remoteID))
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
	dao, err := OpenDb(testDbFile, nil)
	if err != nil {
		t.Error("Couldn't open test.db")
	} else {
		defer removeTestDb(t, dao)
		cbError := errors.New("callback failure")

		err := dao.update(func(tx *boltTx) error {
			return cbError
		})

		if err != cbError {
			t.Error("expected callback error")
		}
		err = dao.db.View(func(tx *bolt.Tx) error {
			if tx.Bucket([]byte(byIDBucket)) != nil {
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

	dao, err := OpenDb(testDbFile, nil)
	if err != nil {
		t.Error("Couldn't open test.db")
	} else {
		defer removeTestDb(t, dao)
		dao.db.Update(func(tx *bolt.Tx) error {
			byPath, _ := tx.CreateBucket([]byte(byPathBucket))
			byPath.Put([]byte(tests[0].path), []byte("fileId"))
			byID, _ := tx.CreateBucket([]byte(byIDBucket))
			byID.Put([]byte("fileId"), toBytes(tests[0].record))
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

func TestBoltDao_FindByID(t *testing.T) {
	tests := []struct {
		description string
		fileId      *filesys.FileID
		record      *RemoteFile
	}{
		{"file exists", &filesys.FileID{"hd1", 1234}, &RemoteFile{}},
		{"file does not exist", &filesys.FileID{"hd1", 5678}, nil},
	}

	dao, err := OpenDb(testDbFile, nil)
	if err != nil {
		t.Error("Couldn't open test.db")
	} else {
		defer removeTestDb(t, dao)
		dao.db.Update(func(tx *bolt.Tx) error {
			byPath, _ := tx.CreateBucket([]byte(byPathBucket))
			byPath.Put([]byte("/remote/path"), []byte(tests[0].fileId.String()))
			byID, _ := tx.CreateBucket([]byte(byIDBucket))
			byID.Put([]byte(tests[0].fileId.String()), toBytes(tests[0].record))
			return nil
		})
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			rf := dao.FindByID(test.fileId)
			if rf == nil && test.record != nil {
				t.Fatal("Expected record but got nil")
			}
		})
	}
}
