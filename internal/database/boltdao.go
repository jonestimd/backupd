package database

import (
	"time"

	bolt "github.com/coreos/bbolt"
)

const (
	byIdBucket   = "FilesById"
	byPathBucket = "FilesByPath"
)

type boltDao struct {
	db *bolt.DB
}

func OpenBoltDb(fileName string) (*boltDao, error) {
	db, err := bolt.Open(fileName, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	return &boltDao{db}, nil
}

func (dao *boltDao) Close() error {
	return dao.db.Close()
}

func (dao *boltDao) IsEmpty() bool {
	var isEmpty bool
	dao.db.View(func(tx *bolt.Tx) error {
		isEmpty = tx.Bucket([]byte(byIdBucket)) == nil
		return nil
	})
	return isEmpty
}

func (dao *boltDao) View(cb func(Transaction) error) error {
	return dao.db.View(func(tx *bolt.Tx) error {
		return cb(&boltTx{tx.Bucket([]byte(byIdBucket)), tx.Bucket([]byte(byPathBucket))})
	})
}

func (dao *boltDao) Update(cb func(Transaction) error) error {
	return dao.db.Update(func(tx *bolt.Tx) error {
		byId, err := tx.CreateBucketIfNotExists([]byte(byIdBucket))
		if err != nil {
			return err
		}
		byPath, err := tx.CreateBucketIfNotExists([]byte(byPathBucket))
		if err != nil {
			return err
		}
		return cb(&boltTx{byId, byPath})
	})
}

func (dao *boltDao) FindByPath(path string) *remoteFile {
	var rf *remoteFile
	dao.db.View(func (tx *bolt.Tx) error {
		if fileId := tx.Bucket([]byte(byPathBucket)).Get([]byte(path)); fileId != nil {
			rf = toRemoteFile(tx.Bucket([]byte(byIdBucket)).Get(fileId))
		}
		return nil
	})
	return rf
}
