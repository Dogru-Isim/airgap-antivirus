package main

import (
	"context"
	"fmt"
	"github.com/Dogru-Isim/airgap-antivirus/internal/config"
	"github.com/Dogru-Isim/airgap-antivirus/internal/logging"
	"github.com/Dogru-Isim/airgap-antivirus/internal/monitoring"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Create a context that cancels on interrupt signals
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, // ^C
		syscall.SIGTERM,
	)
	defer stop()

	if err := run(ctx); err != nil {
		log.Printf("Error: %v", err)
		os.Exit(1)
	}
	log.Println("Shutdown complete")
}

func run(ctx context.Context) error {
	appConfig, err := config.Load()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	log.Printf("Version: %s", appConfig.Version)

	cpuLogger, err := logging.GetLoggerUsingConfig()
	if err != nil {
		log.Fatalf("logging.GetLoggerUsingConfig() failed: %s", err)
	}
	cpuMonitor, err := monitoring.NewCPUMonitor(
		5,                                      // windowSize
		monitoring.WithInterval(1*time.Second), // interval
		monitoring.WithLogger(&cpuLogger),
	)
	if err != nil {
		return fmt.Errorf("cpu monitoring init failed: %w", err)
	}

	// TODO: Move this functionality info CPUMonitor.Start()
	ticker := time.NewTicker(cpuMonitor.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down gracefully")
			return nil
		case <-ticker.C:
			cpuInfo, err := cpuMonitor.GetCPUInfo()
			cpuMonitor.Sync.Do(func() {
				log.Printf("Number of logical cores: %d", cpuInfo.LogicalCores)
			})
			if err := cpuMonitor.CollectMetrics(); err != nil {
				return fmt.Errorf("cpu monitoring error: %w", err)
			}
			if err != nil {
				return fmt.Errorf("cpu monitoring error: %w", err)
			}
		}
	}
}
