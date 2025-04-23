package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	Version string `yaml:"version"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(filepath.Join("configs", path))
	if err != nil {
		return nil, fmt.Errorf("config read error: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config parse error: %w", err)
	}

	return &cfg, nil
}
