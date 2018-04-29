package database

import (
	"bytes"
	"encoding/gob"
	"time"
)

// Cache record for a remote file.
type RemoteFile struct {
	Name         string
	MimeType     string
	Size         uint64
	Md5Checksum  *string
	ParentIDs    []string // remote IDs of the file's parents
	LastModified *string
	LocalID      *string
	RemoteID     *string
}

func NewRemoteFile(name string, mimeType string, size uint64, md5Checksum string, parentIDs []string, modifiedTime string, localID string, remoteID string) *RemoteFile {
	return &RemoteFile{name, mimeType, size, &md5Checksum, parentIDs, &modifiedTime, &localID, &remoteID}
}

func toRemoteFile(b []byte) *RemoteFile {
	rf := RemoteFile{}
	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&rf); err != nil {
		panic(err)
	}
	return &rf
}

func toBytes(rf *RemoteFile) []byte {
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(rf); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func (rf *RemoteFile) ModTime() time.Time {
	t, err := time.Parse(time.RFC3339, *rf.LastModified)
	if err != nil {
		return time.Time{}
	}
	return t
}
