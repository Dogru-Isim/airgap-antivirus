package config

import (
	"fmt"
        "gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"sync"
)

type AppConfig struct {
	ExecutableLocation    string // populated programmatically instead of from a config file
	Version               string `yaml:"version"`
	LogPath               string `yaml:"log_path"`
	CPULogger             string `yaml:"cpu_logger"`
	CPUMonitoringInterval int64  `yaml:"cpu_monitoring_interval"` // type is casted to time.Duration therefore it's stored as int64
	USBLogger             string `yaml:"usb_logger"`
}

var lock = &sync.Mutex{}

// Singleton
var appConfig *AppConfig
var syncOnce sync.Once

func Load() *AppConfig {
	syncOnce.Do(func() {
		executableLocation, err := os.Executable()
		if err != nil {
			fmt.Printf("executable source directory read error: %v\n", err)
			os.Exit(1)
		}
		appConfigData, err := os.ReadFile(filepath.Join(executableLocation, "../../../configs/config.yaml")) // cmd/build/<executable_name>/../../../configs/config.yaml
		//appConfigData, err := os.ReadFile(filepath.Join(executableLocation, "../../configs/config.yaml")) // cmd/build/<executable_name>/../../configs/config.yaml
		//appConfigData, err := os.ReadFile("/home/mint/University/OpenLearning/GroupProjects/AirGappedSystemSecurity/Github/air-gapped/software-based-detection/antivirus-project/configs/config.yaml")
		if err != nil {
			fmt.Printf("config read error: %v\n", err)
			os.Exit(1)
		}

		if err := yaml.Unmarshal(appConfigData, &appConfig); err != nil {
			fmt.Printf("config parse error: %v\n", err)
			os.Exit(1)
		}
		appConfig.ExecutableLocation = executableLocation
	})

	return appConfig
}
