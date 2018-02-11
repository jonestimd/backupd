package config

import (
	"io/ioutil"

	"github.com/go-yaml/yaml"
	"errors"
)

const (
	GoogleDriveName = "googleDrive"
)

type Backend struct {
	Type   string
	Config map[string]*string
}

type Destination struct {
	Backend *string
	Folder  *string
	Encrypt bool
}

type Source struct {
	Path        *string
	Destination *Destination
}

type Config struct {
	Backends map[string]*Backend
	Sources  []*Source
}

func Parse(filename string) (*Config, error) {
	buffer, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err = yaml.Unmarshal(buffer, &cfg); err != nil {
		return nil, err
	}
	for _, source := range cfg.Sources {
		if cfg.Backends[*source.Destination.Backend] == nil {
			return nil, errors.New("Backend not configured: " + *source.Destination.Backend)
		}
	}
	return &cfg, nil
}