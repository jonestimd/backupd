package backend

import (
	"log"
	"os"

	"path/filepath"
	"time"

	"github.com/jonestimd/backupd/internal/config"
	"github.com/jonestimd/backupd/internal/database"
	"github.com/jonestimd/backupd/internal/filesys"
)

// backupService is the interface for reading and writing remote files/directories.
type backupService interface {
	loadFiles() (chan database.FileOrError, error)
	//store(localPath *string, fileID *filesys.FileID, dest *Destination)
	//update(localPath *string)
	//move(newLocalPath *string, rf *database.RemoteFile)
	//trash(localPath *string)
}

// A backend represents a backup storage location.  A backend may be associated with multiple local directories.
type backend struct {
	queue *Queue            // pending updates
	cache *database.BoltDao // Bolt database of backup state
	srv   backupService     // Google Drive, etc.
}

type serviceFactory func(configDir *string, dataDir *string, cfg *config.Backend) (backupService, error)

var serviceFactories = map[string]serviceFactory{
	config.GoogleDriveName: func(configDir *string, dataDir *string, cfg *config.Backend) (backupService, error) {
		return newGoogleDrive(configDir, dataDir, cfg)
	},
}

var defaultDataFile = map[string]string{
	config.GoogleDriveName: "googleDrive.db",
}

// Connect initializes the backends.
func Connect(configDir *string, dataDir *string, backupConfig *config.Config) []*Destination {
	backends := make(map[string]*backend)
	for name, cfg := range backupConfig.Backends {
		factory := serviceFactories[cfg.Type]
		if factory != nil {
			srv, err := factory(configDir, dataDir, cfg)
			if err != nil {
				panic(err)
			}
			backends[name] = newBackend(srv, dataDir, cfg)
		} else {
			log.Println("Unknown destination type: " + cfg.Type)
		}
	}
	dests := make([]*Destination, len(backupConfig.Sources))
	for i, s := range backupConfig.Sources {
		dests[i] = newDestination(backends[*s.Destination.Backend], s.Path, s.Destination.Folder, s.Destination.Encrypt)
	}
	return dests
}

func newBackend(srv backupService, dataDir *string, cfg *config.Backend) *backend {
	dataFile := filepath.Join(*dataDir, cfg.GetParameter("dataFile", defaultDataFile[cfg.Type]))
	cache, err := database.OpenDb(dataFile, srv.loadFiles)
	if err != nil {
		panic(err)
	}
	return &backend{queue: NewQueue(), cache: cache, srv: srv}
}

func (b *backend) processQueue() {
	// TODO handle shutdown
	for {
		m := b.queue.Get()
		switch m.action {
		case StoreAction:
			if fileID, err := filesys.Stat(*m.local); err == nil {
				if rf := b.cache.FindByID(fileID); rf != nil {
					//service.move(m.local, rf)
				} else {
					//service.store(m.local, fileID, m.dest)
				}
			} else {
				log.Printf("Can't stat %s\n", *m.local)
			}
		case UpdateAction:
			//service.update(m.local)
		case TrashAction:
			//service.trash(m.local)
		}
	}
}

// Checks the status of the file and adds it to the backup queue if it has changed or if it has never been backed up.
// Used for startup.
func (b *backend) Init(localPath string, remotePath string) {
	rf := b.cache.FindByPath(remotePath)
	if rf == nil { // TODO verify local file still exists?
		b.queue.Add(&Message{&localPath, &remotePath, StoreAction})
	} else {
		info, err := os.Stat(localPath)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Fatalf("Error getting status of %s: %v\n", localPath, err)
			}
		} else {
			// TODO check checksum?  don't check mod time?
			if uint64(info.Size()) != rf.Size || info.ModTime().Format(time.RFC3339) > *rf.LastModified {
				b.queue.Add(&Message{&localPath, &remotePath, UpdateAction})
			}
		}
	}
}
