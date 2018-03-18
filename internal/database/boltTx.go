package database

import (
	"path/filepath"
)

type boltTx struct {
	byRemoteId   bucket
	byRemotePath bucket
}

func (tx *boltTx) InsertFile(remoteId string, name string, mimeType string, size uint64, md5checksum *string,
							 parentIds []string, lastModified string, localId *string) error {
	rf := RemoteFile{name, mimeType, size, md5checksum, parentIds, &lastModified, localId, &remoteId}
	return tx.byRemoteId.Put([]byte(remoteId), toBytes(&rf))
}

func (tx *boltTx) SetPaths() error {
	return tx.byRemoteId.ForEach(func(id, value []byte) error {
		paths := getPaths(tx.byRemoteId, string(id))
		for _, path := range paths {
			if err := tx.byRemotePath.Put([]byte(path), id); err != nil {
				return err
			}
		}
		return nil
	})
}

func (tx *boltTx) ForEachPath(cb func(path string, fileId string) error) error {
	return tx.byRemotePath.ForEach(func(key []byte, value []byte) error {
		return cb(string(key), string(value))
	})
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
		currentPath := stack[0]
		stack = stack[1:]
		file = getFile(byId, currentPath.nextId)
		if file == nil {
			paths = append(paths, currentPath.String())
		} else if len(file.ParentIds) == 0 {
			paths = append(paths, currentPath.append(file.Name, nil).String())
		} else {
			for i := 0; i < len(file.ParentIds); i++ {
				stack = append(stack, currentPath.append(file.Name, &file.ParentIds[i]))
			}
		}
	}
	return paths
}

func getFile(b bucket, key *string) *RemoteFile {
	buf := b.Get([]byte(*key))
	if buf == nil {
		return nil
	}
	return toRemoteFile(buf)
}
