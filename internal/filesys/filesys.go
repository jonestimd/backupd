package filesys

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// FileInfo contains information about a local file.
type FileInfo struct {
	fsID string
	ino  uint64
	size uint64
}

// ID returns a unique identifier for the file.
func (info *FileInfo) ID() string {
	return fmt.Sprintf("%s-%016x", info.fsID, info.ino)
}

// Size returns the size of the file in bytes.
func (info *FileInfo) Size() uint64 {
	return info.size
}

// ListDirectories writes directories starting with path to the provided channel.
func ListDirectories(path string, ch chan string) {
	stat, err := os.Lstat(path)
	if err != nil {
		log.Print(err)
	} else {
		if stat.Mode().IsDir() {
			ch <- path
			stack := make([]string, 0, 1)
			stack = append(stack, path)
			for len(stack) > 0 {
				path = stack[0]
				stack = stack[1:]
				dirs, err := listDirs(path, ch)
				if err != nil {
					log.Fatal(err)
				}
				stack = append(stack, dirs...)
			}
		}
	}
	close(ch)
}

func listDirs(path string, ch chan string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	infos, err := file.Readdir(-1)
	if err != nil {
		return nil, err
	}
	dirs := make([]string, 0, len(infos))
	for i := 0; i < len(infos); i++ {
		if infos[i].IsDir() {
			child := filepath.Join(path, infos[i].Name())
			dirs = append(dirs, child)
			ch <- child
		}
	}
	return dirs, nil
}
