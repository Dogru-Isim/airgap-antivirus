package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	Version   string `yaml:"version"`
	CPULogger string `yaml:"cpu_logger"`
}

type ConstantsConfig struct {
	Cpu_Logger struct {
		Pretty string `yaml:"pretty"`
		Json   string `yaml:"json"`
	}
}

func Load() (*AppConfig, *ConstantsConfig, error) {
	executableLocation, err := os.Executable()
	if err != nil {
		return nil, nil, fmt.Errorf("executable source directory read error: %w", err)
	}
	appConfigData, err := os.ReadFile(filepath.Join(executableLocation, "../../../configs/config.yaml"))
	if err != nil {
		return nil, nil, fmt.Errorf("config read error: %w", err)
	}

	constantsData, err := os.ReadFile(filepath.Join(executableLocation, "../../../configs/constants.yaml"))
	if err != nil {
		return nil, nil, fmt.Errorf("config read error: %w", err)
	}

	var appConfig AppConfig
	var constantsConfig ConstantsConfig

	if err := yaml.Unmarshal(appConfigData, &appConfig); err != nil {
		return nil, nil, fmt.Errorf("config parse error: %w", err)
	}

	if err := yaml.Unmarshal(constantsData, &constantsConfig); err != nil {
		return nil, nil, fmt.Errorf("config parse error: %w", err)
	}

	return &appConfig, &constantsConfig, nil
}
