package filesys

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// Stat returns information about a local file.
func Stat(path string) (*FileInfo, error) {
	var finfo unix.Stat_t
	if err := unix.Stat(path, &finfo); err != nil {
		return nil, err
	}
	var fsinfo unix.Statfs_t
	if err := unix.Statfs(path, &fsinfo); err != nil {
		return nil, err
	}
	fsID := fmt.Sprintf("%08x%08x", uint32(fsinfo.Fsid.X__val[0]), uint32(fsinfo.Fsid.X__val[1]))
	return &FileInfo{fsID, finfo.Ino, uint64(finfo.Size)}, nil
}
