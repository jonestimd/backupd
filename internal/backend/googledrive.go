package backend

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jonestimd/backupd/internal/config"
	"github.com/jonestimd/backupd/internal/database"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

const (
	defaultDataFile       = "google_drive.db"
	defaultSecretFile     = "gd_client_secret.json"
	defaultTokenFile      = "gd_token.json"
	defaultFolderMimeType = "application/vnd.google-apps.folder"
	defaultRootFolderId   = "root"
	fileFields            = "nextPageToken, files(id, name, parents, mimeType, md5Checksum, size, modifiedTime, trashed, shared, version)"
)

type googleDrive struct {
	destinationFolder string
	folderMimeType    string
	rootFolderId      string
	srv               *drive.Service
	dao               database.Dao
}

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(tokenFile string, ctx context.Context, config *oauth2.Config) *http.Client {
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
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
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

// Create a connection to Google Drive
func NewGoogleDrive(configDir *string, dataDir *string, dest *config.Destination) (*googleDrive, error) {
	backend := googleDrive{
		folderMimeType:    getParameter(dest.Config, "folderMimeType", defaultFolderMimeType),
		rootFolderId:      getParameter(dest.Config, "rootFolderId", defaultRootFolderId),
		destinationFolder: dest.Folder,
	}

	if err := backend.connect(configDir, dataDir, dest); err != nil {
		return nil, err
	}

	return &backend, backend.openCache(filepath.Join(*dataDir, getParameter(dest.Config, "dataFile", defaultDataFile)))
}

// Connect to google drive.
func (gd *googleDrive) connect(configDir *string, dataDir *string, dest *config.Destination) error {
	tokenFile := tokenCacheFile(dataDir, getParameter(dest.Config, "tokenFile", defaultTokenFile))

	ctx := context.Background()

	clientSecretFile := getParameter(dest.Config, "clientSecretFile", defaultSecretFile)
	b, err := ioutil.ReadFile(filepath.Join(*configDir, clientSecretFile))
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
		return err
	}

	// If modifying these scopes, delete your previously saved credentials
	// at ~/.backupd/gd_token.json
	oauthConfig, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
		return err
	}
	client := getClient(tokenFile, ctx, oauthConfig)

	gd.srv, err = drive.New(client)
	if err != nil {
		log.Fatalf("Unable to create drive Client %v", err)
		return err
	}
	return nil
}

// Initialize the file info cache
func (gd *googleDrive) openCache(dataFile string) (err error) {
	gd.dao, err = database.OpenBoltDb(dataFile)
	if err != nil {
		log.Fatalf("Failed to open database: %v\n", err)
		return err
	}
	if gd.dao.IsEmpty() {
		log.Println("Getting files from Google Drive")
		return gd.dao.Update(gd.loadFiles)
	}
	return nil
}

func (gd *googleDrive) loadFiles(tx database.Transaction) (err error) {
	err = gd.listFiles(func(page *drive.FileList) error {
		for _, f := range page.Files {
			if err := insertFile(tx, f); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return tx.SetPaths()
}

func insertFile(tx database.Transaction, f *drive.File) error {
	if !f.Shared {
		return tx.InsertFile(f.Id, f.Name, uint64(f.Size), &f.Md5Checksum, f.Parents, f.ModifiedTime, nil)
	}
	return nil
}

func (gd *googleDrive) listFiles(cb func(*drive.FileList) error) error {
	return gd.srv.Files.List().Fields(fileFields).OrderBy("folder").Q("not trashed").Pages(nil, cb)
}

func (gd *googleDrive) ListFiles() {
	gd.dao.View(func(tx database.Transaction) error {
		tx.ForEachPath(func(path string, fileId string) error {
			fmt.Println(path)
			return nil
		})
		return nil
	})
}
