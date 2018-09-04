package backend

import (
	"sync"
	"testing"

	"os"
	"path/filepath"
	"time"

	"github.com/jonestimd/backupd/internal/config"
	"github.com/jonestimd/backupd/internal/database"
	"github.com/stretchr/testify/assert"
)

const NanosPerSecond = 1000000000

type mockService struct {
	configDir *string
	dataDir   *string
	cfg       *config.Backend
}

type testFile struct {
	size         uint64
	lastModified string
}

func loadFiles() (chan database.FileOrError, error) {
	var ch = make(chan database.FileOrError)
	defer func() {
		close(ch)
	}()
	return ch, nil
}

func (ms *mockService) loadFiles() (chan database.FileOrError, error) {
	return loadFiles()
}

func newTestFile(stat os.FileInfo, offset int64, sizeDelta int64) *testFile {
	modTime := stat.ModTime().Add(time.Duration(offset) * NanosPerSecond).Format(time.RFC3339)
	return &testFile{size: uint64(stat.Size() + sizeDelta), lastModified: modTime}
}

func mockServiceFactory(configDir *string, dataDir *string, cfg *config.Backend) (backupService, error) {
	return &mockService{configDir: configDir, dataDir: dataDir, cfg: cfg}, nil
}

func configuration(backendName string, sourceDir string, destDir string) *config.Config {
	return &config.Config{
		Backends: map[string]*config.Backend{backendName: {Type: config.GoogleDriveName}},
		Sources:  []*config.Source{{Path: &sourceDir, Destination: &config.Destination{Backend: &backendName, Folder: &destDir, Encrypt: false}}},
	}
}

func TestConnect(t *testing.T) {
	originalFactory := serviceFactories[config.GoogleDriveName]
	defer func() {
		serviceFactories[config.GoogleDriveName] = originalFactory
		os.Remove(filepath.Join("testdata", defaultDataFile[config.GoogleDriveName]))
	}()
	serviceFactories[config.GoogleDriveName] = mockServiceFactory
	cfg := configuration("backend 1", "source dir", "dest dir")
	var wg sync.WaitGroup
	halt := make(chan bool)

	dests := Connect(addrOf("config dir"), addrOf("testdata"), cfg, &wg, halt)

	halt <- true
	if len(dests) != 1 {
		t.Errorf("Expected 1 destination, got %d", len(dests))
	} else {
		dests[0].backend.cache.Close()
		srv, ok := dests[0].backend.srv.(*mockService)
		assert.True(t, ok, "Expected mockService")
		assert.Equal(t, "config dir", *srv.configDir)
		assert.Equal(t, "testdata", *srv.dataDir)
		assert.Equal(t, cfg.Backends["backend 1"], srv.cfg)
	}
	wg.Wait()
}

var dbPath = filepath.Join("testdata", "test.db")

func initCache() *database.BoltDao {
	db, _ := database.OpenDb(dbPath, loadFiles)
	return db
}

func initCacheFile(db *database.BoltDao, localPath string, file *testFile) error {
	return db.AddOrUpdate(localPath, localPath, "plain/text", file.size, addrOf("deadbeaf"), nil, file.lastModified, &localPath)
}

func TestBackend_Init(t *testing.T) {
	localFile := filepath.Join("testdata", "to_be_backed_up.txt")
	stat, _ := os.Stat(localFile)
	tests := []struct {
		name      string
		localPath string
		file      *testFile
		count     int
	}{
		{"not backed up", localFile, nil, 1},
		{"backed up, same size and date", localFile, newTestFile(stat, 0, 0), 0},
		{"backed up, older remote file", localFile, newTestFile(stat, -1, 0), 1},
		{"backed up, newer remote file", localFile, newTestFile(stat, 1, 0), 0},
		{"backed up, different size", localFile, newTestFile(stat, 0, 1), 1},
	}

	cache := initCache()
	defer func() {
		cache.Close()
		os.Remove(dbPath)
	}()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.file != nil {
				initCacheFile(cache, test.localPath, test.file)
			}
			b := backend{queue: NewQueue(), cache: cache, srv: &mockService{}}

			b.Init(test.localPath, string(filepath.Separator)+test.localPath)

			assert.Equal(t, test.count, b.queue.items.Len(), "wrong queue length")
		})
	}
}
