package infra

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// SecretConfig matches the structure of secrets/demo.yaml and real.yaml
type SecretConfig struct {
	API struct {
		Bitget struct {
			AccessKey  string `yaml:"access_key"`
			SecretKey  string `yaml:"secret_key"`
			Passphrase string `yaml:"passphrase"`
		} `yaml:"bitget"`
	} `yaml:"api"`
}

// LoadSecretConfig loads API keys from a separate yaml file.
// It returns error if file is missing (Fail Fast).
func LoadSecretConfig(path string) (*SecretConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret config: %w", err)
	}

	var cfg SecretConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse secret config: %w", err)
	}

	return &cfg, nil
}
