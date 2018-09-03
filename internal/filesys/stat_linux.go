package filesys

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func Stat(path string) (key *FileID, err error) {
	var finfo unix.Stat_t
	key = &FileID{}
	if err = unix.Stat(path, &finfo); err != nil {
		return
	}
	key.Ino = finfo.Ino
	var fsinfo unix.Statfs_t
	if err = unix.Statfs(path, &fsinfo); err != nil {
		return
	}
	key.FsID = fmt.Sprintf("%08x%08x", uint32(fsinfo.Fsid.X__val[0]), uint32(fsinfo.Fsid.X__val[1]))
	return
}
