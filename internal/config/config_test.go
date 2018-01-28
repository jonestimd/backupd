package config

import (
    "path/filepath"
    "reflect"
    "testing"
)

func TestParse(t *testing.T) {
    tests := []struct{
        file string
        expected Config
    }{
        {"minimal.yml",
            Config{"", nil, []Source{{"/home/me/Documents", Destination{"googleDrive", "Backups/me", false}}}}},
        {"datadir.yml",
            Config{"~/.backupd", nil, []Source{{"/home/me/Documents", Destination{"googleDrive", "Backups/me", false}}}}},
        {"encrypt.yml",
            Config{"", nil, []Source{{"/home/me/Documents", Destination{"googleDrive", "Backups/me", true}}}}},
        {"googleDriveDefault.yml",
            Config{"", &GoogleDrive{"gd_client_secret.json", "gd_token.json", "", ""}, []Source(nil)}},
        {"googleDrive.yml",
            Config{"", &GoogleDrive{"gd_client_secret.json", "gd_token.json", "application/vnd-custom", "customRoot"}, []Source(nil)}},
    }

    for _, test := range tests {
        t.Run(test.file, func(t *testing.T) {
            actual, err := Parse(filepath.Join("testdata", test.file))
            if err != nil {
                t.Errorf("Unexpected error: %v", err)
            }

            if actual.DataDir != test.expected.DataDir {
                t.Errorf("Expected %s to equal %s", actual.DataDir, test.expected.DataDir)
            }
            if !reflect.DeepEqual(actual.Sources, test.expected.Sources) {
                t.Errorf("Expected %v to equal %v", actual.Sources, test.expected.Sources)
            }
            if !reflect.DeepEqual(actual.GoogleDrive, test.expected.GoogleDrive) {
                t.Errorf("Expected %v to equal %v", actual.GoogleDrive, test.expected.GoogleDrive)
            }
        })
    }
}
