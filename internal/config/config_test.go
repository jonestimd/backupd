package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"github.com/go-yaml/yaml"
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

func TestParse(t *testing.T) {
	tests := []struct {
		file     string
		expected Config
		isError  func (error) bool
	}{
		{"minimal.yml", Config{[]Source{{"/home/me/Documents", Destination{"googleDrive", "Backups/me", false, nil}}}}, nil},
		{"encrypt.yml", Config{[]Source{{"/home/me/Documents", Destination{"googleDrive", "Backups/me", true, nil}}}}, nil},
		{"destinationConfig.yml",
			Config{[]Source{{"/home/me/Documents", Destination{"googleDrive", "Backups/me", false,
				map[string]*string{"clientConfig": addrOf("gd_client_secret.json")}}}}}, nil},
		{"no file", Config{}, os.IsNotExist},
		{"invalid.yml", Config{}, isYamlError},
	}

	for _, test := range tests {
		t.Run(test.file, func(t *testing.T) {
			actual, err := Parse(filepath.Join("testdata", test.file))
			if test.isError == nil {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !reflect.DeepEqual(actual.Sources, test.expected.Sources) {
					t.Errorf("Expected %v to equal %v", actual.Sources, test.expected.Sources)
				}
				if test.expected.Sources[0].Destination.Config != nil {
					if actual.Sources[0].Destination.Config["random"] != nil {
						t.Errorf("Expected %s to be nil", actual.Sources[0].Destination.Config["random"])
					}
				}
			} else if ! test.isError(err) {
				t.Errorf("Not the expected error for \"%s\": %v", test.file, err)
			}
		})
	}
}
