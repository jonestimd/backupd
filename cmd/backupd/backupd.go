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

func startMonitor(dest *backend.Destination) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Error initializing file watcher for %s\n\t%v\n", dest.Source, err)
		os.Exit(1)
	}
	//defer watcher.Close()
	go handleFileChanges(watcher)

	// TODO look for config files, handle ignored files
	// TODO don't use Walk?  Is it too slow (due to sorting)?
	go filepath.Walk(*dest.Source, func(path string, info os.FileInfo, err error) error {
		// TODO how to handle input err
		if err != nil {
			log.Printf("Error walking %s: %v\n", path, err)
		} else if info.IsDir() {
			err := watcher.Add(path)
			if err != nil {
				log.Fatalf("Error adding watcher: %v\n", err)
			}
		} else if info.Mode().IsRegular() {
			dest.Init(path)
		}
		return nil // TODO return SkipDir for ignored directories
	})
}

func handleFileChanges(watcher *fsnotify.Watcher) {
	for {
		select {
		case event := <-watcher.Events:
			log.Println("event:", event)
			if event.Op & fsnotify.Write == fsnotify.Write {
				log.Println("modified file:", event.Name)
				printKey(event.Name)
			}
			if event.Op & fsnotify.Remove == fsnotify.Remove {
				log.Println("Removed file:", event.Name)
				printKey(event.Name)
			}
			if event.Op & fsnotify.Create == fsnotify.Create {
				log.Println("created file:", event.Name)
				printKey(event.Name)
			}
			if event.Op & fsnotify.Rename == fsnotify.Rename {
				log.Println("Renamed file:", event.Name)
				printKey(event.Name)
			}
		case err := <-watcher.Errors:
			log.Println("error:", err)
		}
	}
}

func printKey(path string) {
	fileId, err := filesys.Stat(path)
	if err != nil {
		log.Println("  can't stat", path)
	} else {
		log.Printf("  %s: %s-%016x", path, fileId.FsId, fileId.Ino)
	}
}

func main() {
	flag.Parse()
	if *help {
		flag.Usage()
		os.Exit(1)
	}

	configPath := filepath.Join(*configDir, configFileName)
	cfg, err := config.Parse(configPath)
	if err != nil {
		log.Fatalf("Error reading configuration from %s\n\t%v\n", configPath, err)
		os.Exit(1)
	}
	if len(cfg.Sources) == 0 {
		log.Print("No source directories, exiting")
		os.Exit(1)
	}

	dests, err := backend.Connect(configDir, dataDir, cfg)
	if err != nil {
		log.Fatalf("Failed to connect: %v\n", err)
		os.Exit(1)
	}
	for _, d := range dests {
		startMonitor(d)
	}

	done := make(chan bool)
	<-done
}