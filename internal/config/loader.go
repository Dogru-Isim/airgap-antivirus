package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type AppConfig struct {
	Version               string `yaml:"version"`
	CPULogger             string `yaml:"cpu_logger"`
	CPUMonitoringInterval int64  `yaml:"cpu_monitoring_interval"` // type is casted to time.Duration therefore it's stored as int64
}

func Load() (*AppConfig, error) {
	executableLocation, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("executable source directory read error: %w", err)
	}
	appConfigData, err := os.ReadFile(filepath.Join(executableLocation, "../../../configs/config.yaml"))
	if err != nil {
		return nil, fmt.Errorf("config read error: %w", err)
	}

	var appConfig AppConfig

	if err := yaml.Unmarshal(appConfigData, &appConfig); err != nil {
		return nil, fmt.Errorf("config parse error: %w", err)
	}

	return &appConfig, nil
}
