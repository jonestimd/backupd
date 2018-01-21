package database

import (
	bolt "github.com/coreos/bbolt"
)

type boltTx struct {
	byId   *bolt.Bucket
	byPath *bolt.Bucket
}

func (tx *boltTx) InsertFile(rf *RemoteFile) error {
	return tx.byId.Put([]byte(rf.Id), toBytes(rf))
}

func (tx *boltTx) SetPaths() error {
	return tx.byId.ForEach(func(id, value []byte) error {
		path := getPath(tx.byId, string(id))
		return tx.byPath.Put([]byte(path), id)
	})
}

func (tx *boltTx) ForEachPath(cb func(path string, fileId string) error) error {
	return tx.byPath.ForEach(func(key []byte, value []byte) error {
		return cb(string(key), string(value))
	})
}
