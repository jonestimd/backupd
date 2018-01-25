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

func getFile(b bucket, key *string) *remoteFile {
	buf := b.Get([]byte(*key))
	if buf == nil {
		return nil
	}
	return toRemoteFile(buf)
}

// Get the full path(s) of a remote file
func getPaths(byId bucket, fileId string) []string {
	paths := make([]string, 0)
	stack := make([]*pathNode, 0, 1)
	file := getFile(byId, &fileId)
	if file != nil {
		if file.ParentIds == nil || len(file.ParentIds) == 0 {
			return []string{string(filepath.Separator) + file.Name}
		}
		for i := 0; i < len(file.ParentIds); i++ {
			stack = append(stack, newPathNode([]string{file.Name}, &file.ParentIds[i]))
		}
	}
	for len(stack) > 0 {
		curentPath := stack[0]
		stack = stack[1:]
		file = getFile(byId, curentPath.nextId)
		if file == nil {
			paths = append(paths, curentPath.String())
		} else if len(file.ParentIds) == 0 {
			paths = append(paths, curentPath.append(file.Name, nil).String())
		} else {
			for i := 0; i < len(file.ParentIds); i++ {
				stack = append(stack, curentPath.append(file.Name, &file.ParentIds[i]))
			}
		}
	}
	return paths
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
