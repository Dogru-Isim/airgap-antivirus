package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type AppConfig struct {
	ExecutableLocation    string // populated programmatically instead of from a config file
	Version               string `yaml:"version"`
	CPULogger             string `yaml:"cpu_logger"`
	CPUMonitoringInterval int64  `yaml:"cpu_monitoring_interval"` // type is casted to time.Duration therefore it's stored as int64
	LogPath               string `yaml:"log_path"`
}

func Load() *AppConfig {
	executableLocation, err := os.Executable()
	if err != nil {
		fmt.Println("executable source directory read error: %w", err)
		os.Exit(1)
	}
	appConfigData, err := os.ReadFile(filepath.Join(executableLocation, "../../../configs/config.yaml")) // cmd/build/<executable_name>/../../../configs/config.yaml
	if err != nil {
		fmt.Println("config read error: %w", err)
		os.Exit(1)
	}

	var appConfig AppConfig

	appConfig.ExecutableLocation = executableLocation
	if err := yaml.Unmarshal(appConfigData, &appConfig); err != nil {
		fmt.Println("config parse error: %w", err)
		os.Exit(1)
	}

	return &appConfig
}
