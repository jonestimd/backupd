// Copyright (c) 2018.  Tim  Jones. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package database provides caching of data for a remote destination.
package database

import (
	"io"
)

// Cache record for a remote file.
type remoteFile struct {
	Name         string
	Size         uint64
	Md5Checksum  *string
	ParentIds    []string
	LastModified *string
	LocalId      *string
}

type Transaction interface {
	InsertFile(id string, name string, size uint64, md5checksum *string, parentIds []string, lastModified string, localId *string) error
	SetPaths() error
	ForEachPath(func(path string, fileId string) error) error
}

type Dao interface {
	IsEmpty() bool
	Update(func(Transaction) error) error
	View(func(Transaction) error) error
	io.Closer
}

type bucket interface {
	Get(id []byte) []byte
	Put(id []byte, value []byte) error
	ForEach(func(key []byte, value []byte) error) error
}
