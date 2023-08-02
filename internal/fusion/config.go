package fusion

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type fusionConfig struct {
	DefaultProfile string                   `json:"default_profile"`
	Profiles       map[string]ProfileConfig `json:"profiles"`
}

type ProfileConfig struct {
	ApiHost string `json:"endpoint"`
	Auth    `json:"auth"`
}

type Auth struct {
	IssuerId           string `json:"issuer_id,omitempty"`
	PrivateKeyFile     string `json:"private_pem_file,omitempty"`
	TokenEndpoint      string `json:"token_endpoint,omitempty"`
	AccessToken        string `json:"access_token,omitempty"`
	PrivateKey         string `json:"private_key,omitempty"`
	PrivateKeyPassword string `json:"private_key_password,omitempty"`
}

var (
	ErrNoRequiredAuthFields   = errors.New("profile does not have required auth fields")
	ErrNoDefaultProfileField  = errors.New("config does not have required field `default_profile`")
	ErrNoProfilesField        = errors.New("config does not have required field `profiles`")
	ErrNoProfileExists        = errors.New("profile does not exist")
	ErrNoProfileEndpointField = errors.New("profile does not have required field `endpoint`")
)

func GetProfileConfig(path string, profileName string) (ProfileConfig, error) {
	config, err := readFusionConfig(path)
	if err != nil {
		return ProfileConfig{}, fmt.Errorf("cannot read fusion config: %s", err)
	}

	if profileName == "" {
		profileName = config.DefaultProfile
	}
	if profileName == "" {
		return ProfileConfig{}, ErrNoDefaultProfileField
	}

	return parseProfile(config, profileName)
}

func GetHomeConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configPath := filepath.Join(homeDir, ".pure", "fusion.json")
	return configPath, err
}

func readFusionConfig(path string) (fusionConfig, error) {
	var config fusionConfig

	file, err := os.Open(path)
	if err != nil {
		return fusionConfig{}, err
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		return fusionConfig{}, err
	}
	if config.Profiles == nil {
		return fusionConfig{}, ErrNoProfilesField
	}

	return config, nil
}

func parseProfile(config fusionConfig, profileName string) (ProfileConfig, error) {
	if config.Profiles == nil {
		return ProfileConfig{}, ErrNoProfilesField
	}

	profile, ok := config.Profiles[profileName]
	if !ok {
		return ProfileConfig{}, fmt.Errorf("%s. profile name: %s", ErrNoProfileExists, profileName)
	}

	if (profile.IssuerId == "" || profile.PrivateKeyFile == "") && profile.AccessToken == "" && profile.PrivateKey == "" {
		return ProfileConfig{}, ErrNoRequiredAuthFields
	}

	return profile, nil
}
