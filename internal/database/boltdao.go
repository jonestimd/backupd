package database

import (
	"bytes"
	"encoding/gob"
	"path/filepath"
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

func toBytes(rf *remoteFile) []byte {
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(rf); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func toRemoteFile(b []byte) *remoteFile {
	rf := remoteFile{}
	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&rf); err != nil {
		panic(err)
	}
	return &rf
}

func getFile(b bucket, key string) *remoteFile {
	buf := b.Get([]byte(key))
	if buf == nil {
		return nil
	}
	return toRemoteFile(buf)
}

// Get the full path of a remote file
func getPath(byId bucket, fileId string) string {
	names := make([]string, 0)
	// TODO get all paths for the file
	for file := getFile(byId, fileId); file != nil && len(file.ParentIds) > 0; file = getFile(byId, file.ParentIds[0]) {
		names = append(names, file.Name)
	}
	if len(names) == 0 {
		return string(filepath.Separator)
	}
	for i, j := 0, len(names)-1; i < j; i, j = i+1, j-1 {
		names[i], names[j] = names[j], names[i]
	}
	return string(filepath.Separator) + filepath.Join(names...)
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
		x := &boltTx{byId, byPath}
		return cb(x)
	})
}
