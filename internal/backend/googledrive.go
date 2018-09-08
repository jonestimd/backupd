package backend

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/jonestimd/backupd/internal/config"
	"github.com/jonestimd/backupd/internal/database"
	"github.com/jonestimd/backupd/internal/filesys"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

const (
	defaultSecretFile     = "gd_client_secret.json"
	defaultTokenFile      = "gd_token.json"
	defaultFolderMimeType = "application/vnd.google-apps.folder"
	defaultRootFolderID   = "root"
	fileFields            = "nextPageToken, files(id, name, parents, mimeType, md5Checksum, size, modifiedTime, trashed, shared, version)"
)

/* for mocking in tests */
var configFromJSON = google.ConfigFromJSON
var newDrive = drive.New

// GoogleDrive provides backup to Google Drive.
type GoogleDrive struct {
	folderMimeType string
	rootFolderID   string
	srv            *drive.Service
	listFiles      func(cb func(*drive.FileList) error) error
}

// PathMapper converts between local and remote file paths.
type PathMapper interface {
	RemotePath(localPath string) string
	LocalPath(remotePath string) string
}

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, tokenFile string, config *oauth2.Config) *http.Client {
	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokenFile, tok)
	}
	return config.Client(ctx, tok)
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	log.Printf("Looking for token in %s\n", file)
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	return t, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// Generates a credential file path/filename.  Creates the path if it does not exist.
// Returns the generated path/filename.
func tokenCacheFile(dataDir *string, tokenFile string) string {
	tokenCacheDir := filepath.Join(*dataDir, ".auth")
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir, tokenFile)
}

// Create a connection to Google Drive
func newGoogleDrive(configDir *string, dataDir *string, cfg *config.Backend) (*GoogleDrive, error) {
	gd := &GoogleDrive{
		folderMimeType: cfg.GetParameter("folderMimeType", defaultFolderMimeType),
		rootFolderID:   cfg.GetParameter("rootFolderId", defaultRootFolderID),
	}
	if err := gd.connect(configDir, dataDir, cfg); err != nil {
		return nil, err
	}
	return gd, nil
}

// Connect to google drive.
func (gd *GoogleDrive) connect(configDir *string, dataDir *string, cfg *config.Backend) error {
	tokenFile := tokenCacheFile(dataDir, cfg.GetParameter("tokenFile", defaultTokenFile))

	ctx := context.Background()

	clientSecretFile := cfg.GetParameter("clientSecretFile", defaultSecretFile)
	csBytes, err := ioutil.ReadFile(filepath.Join(*configDir, clientSecretFile))
	if err != nil {
		log.Printf("Unable to read client secret file: %v", err)
		return err
	}

	// If modifying these scopes, delete your previously saved credentials
	// at ~/.backupd/gd_token.json
	oauthConfig, err := configFromJSON(csBytes, drive.DriveScope)
	if err != nil {
		log.Printf("Unable to parse client secret file to config: %v", err)
		return err
	}
	client := getClient(ctx, tokenFile, oauthConfig)

	gd.srv, err = newDrive(client)
	if err != nil {
		log.Printf("Unable to create drive Client %v", err)
		return err
	}
	gd.listFiles = func(cb func(*drive.FileList) error) error {
		return gd.srv.Files.List().Fields(fileFields).OrderBy("folder").Q("not trashed").Pages(nil, cb)
	}
	return nil
}

// loadFiles gets names and properties of all files in the backup location.
func (gd *GoogleDrive) loadFiles() (chan database.FileOrError, error) {
	fileCh := make(chan database.FileOrError)
	go gd.listFiles(func(page *drive.FileList) error {
		for _, f := range page.Files {
			if !f.Shared {
				fileCh <- database.FileOrError{File: &database.RemoteFile{
					RemoteID:     &f.Id,
					Name:         f.Name,
					MimeType:     f.MimeType,
					Size:         uint64(f.Size),
					Md5Checksum:  &f.Md5Checksum,
					ParentIDs:    f.Parents,
					LastModified: &f.ModifiedTime,
				}}
			}
		}
		if page.NextPageToken == "" {
			close(fileCh)
		}
		return nil
	})
	return fileCh, nil
}

// Backup a new file.
func (gd *GoogleDrive) store(localPath *string, finfo *filesys.FileInfo, remotePath *string) {
	log.Printf("Store %s\n", *localPath)
	_, file := path.Split(*remotePath)
	log.Print(file)
}

// Update the backup for an existing file.
func (gd *GoogleDrive) update(localPath *string) {
	log.Printf("Update %s\n", *localPath)
}

// Update the location and/or name of a file.
func (gd *GoogleDrive) move(localPath *string, rf *database.RemoteFile) {
	log.Printf("Move %s to %s\n", *localPath, rf.Name)
}

// Move a backup to the trash folder.
func (gd *GoogleDrive) trash(localPath *string) {
	log.Printf("Trash %s\n", *localPath)
}
