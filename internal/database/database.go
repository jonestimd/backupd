// Copyright (c) 2018.  Tim  Jones. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package database provides caching of data for a remote destination.
// Cached info can be retrieved by local file ID or by remote path.
package database

import (
	"io"
	"github.com/jonestimd/backupd/internal/filesys"
)

type Transaction interface {
	InsertFile(remoteId string, mimeType string, name string, size uint64, md5checksum *string, parentIds []string, lastModified string, localId *string) error
	SetPaths() error
	ForEachPath(func(path string, fileId string) error) error
}

type Dao interface {
	IsEmpty() bool
	Update(func(Transaction) error) error
	View(func(Transaction) error) error
	FindByPath(remotePath string) *RemoteFile
	FindById(id *filesys.FileId) *RemoteFile
	io.Closer
}

type bucket interface {
	Get(id []byte) []byte
	Put(id []byte, value []byte) error
	ForEach(func(key []byte, value []byte) error) error
}
