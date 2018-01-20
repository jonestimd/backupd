package filesys

import (
	"log"
	"os"
	"path/filepath"
)

type FileId struct {
	FsId string
	Ino  uint64
}

func ListDirectories(path string, ch chan string) {
	stat, err := os.Lstat(path)
	if err != nil {
		log.Fatal(err)
	}
	if stat.Mode().IsDir() {
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
	close(ch)
}

func listDirs(path string, ch chan string) ([]string, error) {
	ch <- path
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
