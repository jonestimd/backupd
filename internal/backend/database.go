package backend

import (
	"path/filepath"
	"time"

	bolt "github.com/coreos/bbolt"
)

func openDb(fileName string) (db *bolt.DB, err error) {
	return bolt.Open(fileName, 0600, &bolt.Options{Timeout: 1 * time.Second})
}

func getFile(b *bolt.Bucket, key string) *RemoteFile {
	buf := b.Get([]byte(key))
	if buf == nil {
		return nil
	}
	return toRemoteFile(buf)
}

// Get the full path of a remote file
func getPath(byId *bolt.Bucket, fileId string) string {
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
