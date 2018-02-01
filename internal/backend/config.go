package backend

import (
	"path/filepath"
	"os"
)

func getParameter(config map[string]*string, key string, defaultValue string) string {
	value := config[key]
	if value == nil {
		return defaultValue
	}
	return *value
}

// Generates a credential file path/filename.  Creates the path if it does not exist.
// Returns the generated path/filename.
func tokenCacheFile(dataDir *string, tokenFile string) string {
	tokenCacheDir := filepath.Join(*dataDir, ".auth")
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir, tokenFile)
}
