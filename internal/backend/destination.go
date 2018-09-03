package backend

import (
	"path/filepath"
)

// Destination represents a backup destination for a source folder.  A source folder may have
// multiple backup destinations.
type Destination struct {
	backend    *backend
	LocalRoot  *string
	remoteRoot *string
	encrypt    bool
}

func newDestination(b *backend, localPath *string, remotePath *string, encrypt bool) *Destination {
	return &Destination{backend: b, LocalRoot: localPath, remoteRoot: remotePath, encrypt: encrypt}
}

// Init checks the status of the file and adds it to the backup queue if it has changed or if it has never been backed up.
// Used for startup.
func (d *Destination) Init(localPath string) {
	remotePath := d.RemotePath(localPath)
	d.backend.Init(localPath, remotePath)
}

// RemotePath converts a local path to its corresponding remote path.
func (d *Destination) RemotePath(localPath string) string {
	return filepath.Join(*d.remoteRoot, localPath[len(*d.LocalRoot):])
}

// LocalPath converts a remote path to its corresponding local path.
func (d *Destination) LocalPath(remotePath string) string {
	return filepath.Join(*d.LocalRoot, remotePath[len(*d.remoteRoot):])
}

// Add is called when a new file is created in a watched directory.  Adds the file to the backup queue.
func (d *Destination) Add(localPath string) {
	//d.backend.queue.Add(&message{&localPath, d, StoreAction})
}

// Update is called when a file in a watched directory is modified.  Adds the file to the backup queue.
// Used for content change, rename or move.
func (d *Destination) Update(localPath string) {
	//d.backend.queue.Add(&message{&localPath, d, UpdateAction})
}

// Delete is called when a file is deleted from a watched directory.  Moves the backup copy to the trash folder (maybe).
func (d *Destination) Delete(localPath string) {
	//d.backend.queue.Add(&message{&localPath, d, TrashAction})
}
