package backend

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

const (
	configDir        = ".backupd"
	clientSecretFile = "gd_client_secret.json"
	tokenFile        = "gd_token.json"
	folderMimeType   = "application/vnd.google-apps.folder"
	rootFolderId     = "root"
	fileFields       = "nextPageToken, files(id, name, parents, mimeType, md5Checksum, appProperties, size, modifiedTime, trashed, version)"
)

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile := tokenCacheFile()
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(cacheFile, tok)
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

func getUserHome() string {
	home := os.Getenv("HOME")
	if len(home) == 0 {
		return "~"
	}
	return home
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() string {
	tokenCacheDir := filepath.Join(getUserHome(), configDir)
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir, tokenFile)
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

func listFiles(srv *drive.Service) *drive.FilesListCall {
	return srv.Files.List().Fields(fileFields)
}

func listFilesPage(srv *drive.Service, nextPageToken string) *drive.FilesListCall {
	return listFiles(srv).PageToken(nextPageToken)
}

func getPath(filesById map[string]*drive.File, fileId string) string {
	names := make([]string, 0)
	for file := filesById[fileId]; file != nil && len(file.Parents) > 0; file = filesById[file.Parents[0]] {
		names = append(names, file.Name)
	}
	if len(names) == 0 {
		return ""
	}
	for i, j := 0, len(names)-1; i < j; i, j = i+1, j-1 {
		names[i], names[j] = names[j], names[i]
	}
	return filepath.Join(names...)
}

func ListFiles() error {
	ctx := context.Background()

	b, err := ioutil.ReadFile(filepath.Join(getUserHome(), configDir, clientSecretFile))
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
		return err
	}

	// If modifying these scopes, delete your previously saved credentials
	// at ~/.backupd/gd_token.json
	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
		return err
	}
	client := getClient(ctx, config)

	srv, err := drive.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve drive Client %v", err)
		return err
	}

	filesById := make(map[string]*drive.File)

	for page, err := listFiles(srv).Do(); true; page, err = listFilesPage(srv, page.NextPageToken).Do() {
		if err != nil {
			log.Fatalf("Unable to retrieve files: %v", err)
			return err
		}
		if len(page.Files) > 0 {
			for _, i := range page.Files {
				filesById[i.Id] = i
			}
		}
		if len(page.NextPageToken) == 0 {
			break
		}
	}

	filesByPath := make(map[string]*drive.File)

	for id, file := range filesById {
		filesByPath[getPath(filesById, id)] = file
	}
	for path := range filesByPath {
		log.Println(path)
	}
	return nil
}
