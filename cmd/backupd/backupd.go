package main

import (
	"flag"
	"log"

	"github.com/fsnotify/fsnotify"
	"github.com/jonestimd/backupd/internal/backend"
	"github.com/jonestimd/backupd/internal/config"
	"github.com/jonestimd/backupd/internal/filesys"
	"os"
	"path/filepath"
)

// TODO handle mount/umount for watched directories
// TODO wait for file to be closed before uploading

const (
	configFileName = "backupd.yml"
)

var help = flag.Bool("h", false, "Show help")
var configDir = flag.String("c", defaultConfigDir, "Configuration directory")
var dataDir = flag.String("d", defaultDataDir, "Data directory")

func main() {
	flag.Parse()
	if *help {
		flag.Usage()
		os.Exit(1)
	}

	configPath := filepath.Join(*configDir, configFileName)
	cfg, err := config.Parse(configPath)
	if err != nil {
		log.Fatalf("Error reading configuration from %s: %v\n", configPath, err)
		os.Exit(1)
	}

	for _, source := range cfg.Sources {
		switch source.Destination.Type {
		case config.GoogleDriveName:
			gd, err := backend.NewGoogleDrive(configDir, dataDir, &source.Destination)
			if err != nil {
				log.Fatalf("Error connecting to Google Drive: %v\n", err)
				os.Exit(1)
			}
		}
	}

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
