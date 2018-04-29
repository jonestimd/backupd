package database

import (
	"log"
	"time"

	bolt "github.com/coreos/bbolt"
	"github.com/jonestimd/backupd/internal/filesys"
)

const (
	byIDBucket   = "FilesById"
	byPathBucket = "FilesByPath"
)

type BoltDao struct {
	db *bolt.DB
}

type FileOrError struct {
	File  *RemoteFile
	Error error
}

// Opens the specified data file.  If the database is empty then getFiles is used to populate it.
func OpenDb(fileName string, getFiles func() (chan FileOrError, error)) (*BoltDao, error) {
	db, err := bolt.Open(fileName, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	dao := &BoltDao{db}
	if getFiles != nil && dao.isEmpty() {
		log.Printf("Populating files in %s\n", fileName)
		var ch chan FileOrError
		if ch, err = getFiles(); err != nil {
			dao.Close()
			return nil, err
		}
		err = dao.update(func(tx *boltTx) error {
			for f := range ch {
				if f.Error != nil {
					return f.Error
				}
				if err = tx.byRemoteID.Put([]byte(*f.File.RemoteID), toBytes(f.File)); err != nil {
					return err
				}
			}
			return tx.SetPaths()
		})
		if err != nil {
			dao.Close()
			return nil, err
		}
	}
	return dao, nil
}

func (dao *BoltDao) Close() error {
	return dao.db.Close()
}

func (dao *BoltDao) isEmpty() bool {
	var isEmpty bool
	dao.db.View(func(tx *bolt.Tx) error {
		isEmpty = tx.Bucket([]byte(byIDBucket)) == nil
		return nil
	})
	return isEmpty
}

func (dao *BoltDao) update(cb func(*boltTx) error) error {
	return dao.db.Update(func(tx *bolt.Tx) error {
		byID, err := tx.CreateBucketIfNotExists([]byte(byIDBucket))
		if err != nil {
			return err
		}
		byPath, err := tx.CreateBucketIfNotExists([]byte(byPathBucket))
		if err != nil {
			return err
		}
		return cb(&boltTx{byID, byPath})
	})
}

func (dao *BoltDao) FindByPath(remotePath string) *RemoteFile {
	var rf *RemoteFile
	dao.db.View(func(tx *bolt.Tx) error {
		if fileID := tx.Bucket([]byte(byPathBucket)).Get([]byte(remotePath)); fileID != nil {
			rf = toRemoteFile(tx.Bucket([]byte(byIDBucket)).Get(fileID))
		}
		return nil
	})
	return rf
}

func (dao *BoltDao) FindByID(fileID *filesys.FileID) (rf *RemoteFile) {
	dao.db.View(func(tx *bolt.Tx) error {
		if rec := tx.Bucket([]byte(byIDBucket)).Get([]byte(fileID.String())); rec != nil {
			rf = toRemoteFile(rec)
		}
		return nil
	})
	return
}
