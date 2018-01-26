package filesys

import (
	"testing"
)

func TestListDirectories(t *testing.T) {
	tests := []struct {
		description string
		startPath   string
		expected    []string
	}{
		{"directory", "..", []string{"..", "../database", "../backend", "../filesys", "../database/testdata"}},
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
			if len(result) != len(test.expected) {
				t.Errorf("Expected %d directories, found %d", len(test.expected), len(result))
			}
			for i, d := range result {
				if d != test.expected[i] {
					t.Errorf("Expected path %s to equal %s", d, test.expected[i])
				}
			}
		})
	}
}
