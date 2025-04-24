package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	Version string `yaml:"version"`
}

func Load() (*AppConfig, error) {
	executableLocation, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("executable source directory read error: %w", err)
	}
	data, err := os.ReadFile(filepath.Join(executableLocation, "../../../configs/config.yaml"))
	if err != nil {
		return nil, fmt.Errorf("config read error: %w", err)
	}

	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config parse error: %w", err)
	}

	return &cfg, nil
}
