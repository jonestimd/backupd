package backend

import (
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonestimd/backupd/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
)

type googleMock struct {
	mock.Mock
}

func (mg *googleMock) configFromJSON(jsonkey []byte, scopes ...string) (*oauth2.Config, error) {
	args := mg.Called(jsonkey, scopes)
	return args.Get(0).(*oauth2.Config), args.Error(1)
}

func (mg *googleMock) newDrive(client *http.Client) (*drive.Service, error) {
	args := mg.Called(client)
	return args.Get(0).(*drive.Service), args.Error(1)
}

func TestNewGoogleDrive(t *testing.T) {
	dataDir := "testdata"
	configDir := filepath.Join(dataDir, ".auth")
	badFile := "no_such_file.json"
	tokenFile := "test_token.json"
	jsonkey, _ := ioutil.ReadFile(filepath.Join(dataDir, ".auth", defaultSecretFile))
	authCfgErr := "bad oauth config"
	svcError := "service error"
	tests := []struct {
		name        string
		cfg         *config.Backend
		authCfg     *oauth2.Config
		authCfgErr  error
		svc         *drive.Service
		svcError    error
		expectedErr *string
	}{
		{"error for no client secret file", &config.Backend{Config: map[string]*string{"clientSecretFile": &badFile}},
			nil, nil, nil, nil, addrOf("open testdata/.auth/no_such_file.json: no such file or directory")},
		{"error for oauth config", &config.Backend{Config: map[string]*string{}},
			nil, errors.New(authCfgErr), nil, nil, &authCfgErr},
		{"use saved token", &config.Backend{Config: map[string]*string{}},
			&oauth2.Config{}, nil, nil, nil, nil},
		{"return error from drive.New", &config.Backend{Config: map[string]*string{}},
			&oauth2.Config{}, nil, nil, errors.New(svcError), &svcError},
		// {"get new token", &config.Backend{Config: map[string]*string{"tokenFile": &tokenFile}},
		// 	&oauth2.Config{}, nil, nil, nil, nil},
	}
	defer func() {
		os.Remove(filepath.Join(configDir, tokenFile))
	}()

	for _, test := range tests {
		var mg googleMock
		configFromJSON = mg.configFromJSON
		newDrive = mg.newDrive
		t.Run(test.name, func(t *testing.T) {
			mg.Test(t)
			mg.On("configFromJSON", jsonkey, []string{drive.DriveScope}).Return(test.authCfg, test.authCfgErr)
			mg.On("newDrive", mock.Anything).Return(test.svc, test.svcError)

			gd, err := newGoogleDrive(&configDir, &dataDir, test.cfg)

			if test.expectedErr != nil {
				assert.Equal(t, *test.expectedErr, err.Error())
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, gd)
			}
		})
	}
}
