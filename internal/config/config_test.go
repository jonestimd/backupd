package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/go-yaml/yaml"
	"github.com/stretchr/testify/assert"
)

func addrOf(value string) *string {
	return &value
}

func isYamlError(err error) bool {
	switch err.(type) {
	case *yaml.TypeError:
		return true
	}
	return false
}

func isBadBackend(err error) bool {
	return err.Error() == "Backend not configured: Google Drive"
}

const (
	backendName = "Google Drive"
	backendType = "googleDrive"
)

func newConfig(backends map[string]*Backend, sourcesPath string, destFolder string, encrypt bool) Config {
	dest := &Destination{addrOf(backendName), &destFolder, encrypt}
	source := &Source{&sourcesPath, dest}
	config := Config{backends, []*Source{source}}
	return config
}

func TestParse(t *testing.T) {
	tests := []struct {
		file     string
		expected Config
		isError  func(error) bool
	}{
		{"minimal.yml", newConfig(map[string]*Backend{backendName: {backendType, nil}}, "/home/me/Documents", "Backups/me", false), nil},
		{"encrypt.yml", newConfig(map[string]*Backend{backendName: {backendType, nil}}, "/home/me/Documents", "Backups/me", true), nil},
		{"destinationConfig.yml", newConfig(
			map[string]*Backend{backendName: {backendType, map[string]*string{"clientConfig": addrOf("gd_client_secret.json")}}},
			"/home/me/Documents", "Backups/me", false), nil},
		{"no file", Config{}, os.IsNotExist},
		{"invalid.yml", Config{}, isYamlError},
		{"bad_backend.yml", Config{}, isBadBackend},
	}

	for _, test := range tests {
		t.Run(test.file, func(t *testing.T) {
			actual, err := Parse(filepath.Join("testdata", test.file))
			if test.isError == nil {
				if err != nil {
					t.Errorf("Unexpected error: %#v", err)
				}
				if !reflect.DeepEqual(actual.Backends, test.expected.Backends) {
					t.Errorf("Expected\n%#v to equal\n%#v", actual.Backends, test.expected.Backends)
				}
				if !reflect.DeepEqual(actual.Sources, test.expected.Sources) {
					t.Errorf("Expected\n%#v to equal\n%#v", actual.Sources, test.expected.Sources)
				}
			} else if !test.isError(err) {
				t.Errorf("Not the expected error for \"%s\": %#v", test.file, err)
			}
		})
	}
}

func TestGetParameter(t *testing.T) {
	parameter := "the parameter"
	defaultValue := "the default"
	configValue := "config value"
	tests := []struct {
		name          string
		expectedValue string
		backend       *Backend
	}{
		{"returns default", defaultValue, &Backend{Config: map[string]*string{}}},
		{"returns config value", configValue, &Backend{Config: map[string]*string{parameter: &configValue}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.backend.GetParameter(parameter, defaultValue)

			assert.Equal(t, test.expectedValue, actual)
		})
	}
}
