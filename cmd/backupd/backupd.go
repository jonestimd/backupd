package main

import (
	"flag"
	"log"

	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/jonestimd/backupd/internal/backend"
	"github.com/jonestimd/backupd/internal/config"
	"github.com/jonestimd/backupd/internal/filesys"
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
		log.Fatalf("Error initializing file watcher for %s\n\t%v\n", *dest.LocalRoot, err)
		os.Exit(1)
	}
	//defer watcher.Close()
	go handleFileChanges(watcher, dest)

	// TODO look for config files, handle ignored files
	// TODO don't use Walk?  Is it too slow (due to sorting)?
	go filepath.Walk(*dest.LocalRoot, func(path string, info os.FileInfo, err error) error {
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

func handleFileChanges(watcher *fsnotify.Watcher, dest *backend.Destination) {
	for {
		select {
		case event := <-watcher.Events:
			log.Println("event:", event)
			if (event.Op & fsnotify.Write) == fsnotify.Write { // TODO is file still open?
				dest.Update(event.Name)
				printKey(event.Name)
			}
			if (event.Op & fsnotify.Remove) == fsnotify.Remove {
				dest.Delete(event.Name)
				printKey(event.Name)
			}
			if (event.Op & fsnotify.Create) == fsnotify.Create { // TODO is file still open?
				dest.Add(event.Name)
				printKey(event.Name)
			}
		case err := <-watcher.Errors:
			log.Println("error:", err)
		}
	}
}

func printKey(path string) {
	fileID, err := filesys.Stat(path)
	if err != nil {
		log.Println("  can't stat", path)
	} else {
		log.Printf("  %s: %s-%016x", path, fileID.FsID, fileID.Ino)
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

	dests := backend.Connect(configDir, dataDir, cfg)
	for _, d := range dests {
		// TODO look for deleted files
		startMonitor(d)
	}

	done := make(chan bool)
	<-done
}
