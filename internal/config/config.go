package config

import (
	"io/ioutil"

	"github.com/go-yaml/yaml"
)

const (
	GoogleDriveName = "googleDrive"
)

type DestinationType int

type Destination struct {
	Type    string
	Folder  string
	Encrypt bool
	Config  map[string]*string
}

type Source struct {
	Path        string
	Destination Destination
}

type Config struct {
	Sources []Source
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
