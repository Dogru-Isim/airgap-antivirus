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
	appConfig := config.Load()

	log.Printf("Version: %s", appConfig.Version)

	cpuLogger, err := logging.GetCPULoggerUsingConfig()
	if err != nil {
		log.Fatalf("logging.GetLoggerUsingConfig() failed: %s", err)
	}
	cpuMonitor, err := monitoring.NewCPUMonitor(
		5, // windowSize
		monitoring.WithInterval(time.Duration(appConfig.CPUMonitoringInterval)*time.Millisecond), // interval
		monitoring.WithLogger(&cpuLogger),
	)
	if err != nil {
		return fmt.Errorf("cpu monitoring init failed: %w", err)
	}

	// Create a channel to signal completion
	done := make(chan struct{})

	// Start the CPU monitor in a goroutine
	go func() {
		defer close(done) // Signal that this goroutine is done
		if err := cpuMonitor.Start(ctx); err != nil {
			log.Printf("Error in CPU Monitor: %v", err)
		}
	}()

	// Start the USB monitoring in a goroutine
	go func() {
		defer close(done) // Signal that this goroutine is done
		monitoring.MonitorUSB(ctx)
	}()

	// Wait for both goroutines to finish
	<-done
	<-done // Wait for the second goroutine

	return nil
}
