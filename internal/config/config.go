package config

import (
    "io/ioutil"

    "github.com/go-yaml/yaml"
)

const (
    googleDriveName = "googleDrive"
)

type DestinationType int

const (
    T_GoogleDrive DestinationType = iota
)

type Destination struct {
    Type    string
    Folder  string
    Encrypt bool
}

type Source struct {
    Path        string
    Destination Destination
}

type GoogleDrive struct {
    ClientConfig   string `yaml:"clientConfig"`
    TokenFile      string `yaml:"tokenFile"`
    FolderMimeType string `yaml:"folderMimeType,omitempty"`
    RootFolderId   string `yaml:"rootFolderId,omitempty"`
}

type Config struct {
    DataDir     string `yaml:"dataDir,omitempty"`
    GoogleDrive *GoogleDrive `yaml:"googleDrive,omitempty"`
    Sources     []Source
}

func Parse(filename string) (*Config, error) {
    buffer, err := ioutil.ReadFile(filename)
    if err != nil {
        return nil, err
    }
    var config Config
    if err = yaml.Unmarshal(buffer, &config); err != nil {
        return nil, err
    }
    return &config, nil
}
