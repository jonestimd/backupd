package main

import (
	"flag"
	"log"

	"github.com/fsnotify/fsnotify"
	"github.com/jonestimd/backupd/internal/backend"
	"github.com/jonestimd/backupd/internal/filesys"
)

// TODO handle mount/umount for watched directories
// TODO wait for file to be closed before uploading

func main() {
	flag.Parse()

	backend.ListFiles()

	path := "/home/tim/Documents"

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	dirs := make(chan string)
	go filesys.ListDirectories(path, dirs)
	for d := range dirs {
		log.Println(d)
	}

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
					printKey(event.Name)
				}
				if event.Op&fsnotify.Remove == fsnotify.Remove {
					log.Println("Removed file:", event.Name)
					printKey(event.Name)
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					log.Println("created file:", event.Name)
					printKey(event.Name)
				}
				if event.Op&fsnotify.Rename == fsnotify.Rename {
					log.Println("Renamed file:", event.Name)
					printKey(event.Name)
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(path)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

func printKey(path string) {
	fileId, err := filesys.Stat(path)
	if err != nil {
		log.Println("  can't stat", path)
	} else {
		log.Printf("  %s: %s-%016x", path, fileId.FsId, fileId.Ino)
	}
}
