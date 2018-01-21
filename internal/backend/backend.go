package backend

import (
	"bytes"
	"encoding/gob"
)

type RemoteFile struct {
	Id           string
	Name         string
	Size         uint64
	Md5Checksum  *string
	ParentIds    []string
	LastModified *string
	LocalId      *string
}

func (rf *RemoteFile) toBytes() []byte {
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(rf); err != nil {
		panic(err)
	}
	return buf.Bytes()
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

type Backend interface {
	ListFiles()
}
