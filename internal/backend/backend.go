package backend

import (
	"github.com/jonestimd/backupd/internal/config"
	"errors"
	"github.com/jonestimd/backupd/internal/database"
	"os"
	"log"
)

const (
	defaultDataFile = "backupd.db"
)

type remoteStatus struct {
	Exists       bool
	Size         uint64
	Md5Checksum  *string
	LastModified *string
}

type Backend struct {
	queue *queue
	dao   database.Dao
}

type Destination struct {
	backend *Backend
	Source  *string
	folder  *string
	encrypt bool
}

// Initialize backends.
func Connect(configDir *string, dataDir *string, cfg *config.Config) ([]*Destination, error) {
	backends := make(map[string]*Backend)
	for name, cfg := range cfg.Backends {
		switch cfg.Type {
		case config.GoogleDriveName:
			gd, err := newGoogleDrive(configDir, dataDir, cfg)
			if err != nil {
				return nil, err
			}
			backends[name] = &gd.Backend
		default:
			return nil, errors.New("Unknown destination type: " + cfg.Type)
		}
	}
	dests := make([]*Destination, len(cfg.Sources))
	for i, s := range cfg.Sources {
		dests[i] = &Destination{backends[*s.Destination.Backend], s.Path, s.Destination.Folder, s.Destination.Encrypt}
	}
	return dests, nil
}

func (d *Destination) remotePath(localPath string) string {
	return localPath[len(*d.Source):]
}

// Checks the status of the file and add it to the backup queue if it has changed or if it has never been backed up.
// Used for startup.
func (d *Destination) Init(localPath string) {
	remotePath := d.remotePath(localPath)
	rf := d.backend.dao.FindByPath(remotePath)
	if rf == nil {
		d.backend.queue.Add(&message{&localPath, &remotePath, StoreAction})
	} else {
		info, err := os.Stat(localPath)
		if err != nil {
			if ! os.IsNotExist(err) {
				log.Fatalf("Error getting status of %s: %v\n", localPath, err)
			}
		} else {
			if uint64(info.Size()) != rf.Size || info.ModTime().After(rf.ModTime()) {
				d.backend.queue.Add(&message{&localPath, &remotePath, UpdateAction})
			}
		}
	}
}

// Notification of a new file.  Adds the file to the backup queue.
func (d *Destination) Add(localPath string) {
	remotePath := d.remotePath(localPath)
	d.backend.queue.Add(&message{&localPath, &remotePath, StoreAction})
}

// Notification that the file has been modified.  Adds the file to the backup queue.
// Used for content change, rename or move.
func (d *Destination) Update(localPath string) {
	remotePath := d.remotePath(localPath)
	d.backend.queue.Add(&message{&localPath, &remotePath, UpdateAction})
}

// Notification that the file has been deleted.  Moves the backup copy to the trash folder (maybe).
func (d *Destination) Rename(localPath string, newName string) {
	remotePath := d.remotePath(localPath)
	d.backend.queue.Add(&message{&newName, &remotePath, RenameAction})
}

// Notification that the file has been deleted.  Moves the backup copy to the trash folder (maybe).
func (d *Destination) Move(localPath string, newName string) {
	remotePath := d.remotePath(localPath)
	d.backend.queue.Add(&message{&newName, &remotePath, MoveAction})
}

// Notification that the file has been deleted.  Moves the backup copy to the trash folder (maybe).
func (d *Destination) Delete(localPath string) {
	remotePath := d.remotePath(localPath)
	d.backend.queue.Add(&message{&localPath, &remotePath, TrashAction})
}
