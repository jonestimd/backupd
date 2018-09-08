package filesys

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListDirectories(t *testing.T) {
	tests := []struct {
		description string
		startPath   string
		expected    []string
	}{
		{"directory", "..", []string{
			"..", "../database", "../backend", "../config", "../filesys", "../database/testdata",
			"../backend/testdata", "../config/testdata", "../backend/testdata/.auth"}},
		{"file", "filesys.go", []string{}},
		{"unknown", "x", []string{}},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			dirs := make(chan string)

			go ListDirectories(test.startPath, dirs)

			result := make([]string, 0)
			for d := range dirs {
				result = append(result, d)
			}
			assert.Equal(t, len(test.expected), len(result), "Directory count mismatch")
			if len(result) == len(test.expected) {
				for i, d := range result {
					assert.Equal(t, test.expected[i], d)
				}
			}
		})
	}
}

func TestFileInfo_String(t *testing.T) {
	info := &FileInfo{"file sys ID", 0xdeadbeefabacab, 0}

	if info.ID() != "file sys ID-00deadbeefabacab" {
		t.Errorf("Wrong format for file ID: %s", info.ID())
	}
}
