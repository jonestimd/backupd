package database

type boltTx struct {
	byId   bucket
	byPath bucket
}

func (tx *boltTx) InsertFile(id string, name string, size uint64, md5checksum *string, parentIds []string, lastModified string, localId *string) error {
	rf := remoteFile{name, size, md5checksum, parentIds, &lastModified, localId}
	return tx.byId.Put([]byte(id), toBytes(&rf))
}

func (tx *boltTx) SetPaths() error {
	return tx.byId.ForEach(func(id, value []byte) error {
		paths := getPaths(tx.byId, string(id))
		for _, path := range paths {
			if err := tx.byPath.Put([]byte(path), id); err != nil {
				return err
			}
		}
		return nil
	})
}

func (tx *boltTx) ForEachPath(cb func(path string, fileId string) error) error {
	return tx.byPath.ForEach(func(key []byte, value []byte) error {
		return cb(string(key), string(value))
	})
}
