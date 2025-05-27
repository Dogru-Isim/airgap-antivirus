package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Dogru-Isim/airgap-antivirus/internal/config"
	"github.com/Dogru-Isim/airgap-antivirus/internal/logging"
	"github.com/Dogru-Isim/airgap-antivirus/internal/monitoring"
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

	usbDetector := monitoring.NewUSBDetector()
	// Create a channel to signal completion
	done := make(chan struct{}, 2)

	// Start the CPU monitor in a goroutine
	go func() {
		defer close(done) // Signal that this goroutine is done
		if err := cpuMonitor.Start(ctx); err != nil {
			log.Printf("Error in CPU Monitor: %v", err)
		}
	}()

	// Start the USB monitoring in a goroutine
	go func() {
		// defer close(done)         // Signal that this goroutine is done
		// monitoring.DetectingUSB() // give context
		defer func() { done <- struct{}{} }()
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// USB detectie logica

				if err := usbDetector.DetectNewUSB(); err != nil {
					log.Printf("USB detection error: %v", err)
					continue
				}
				usbDetector.USBDifferenceChecker()

				if usbDetector.NewUSB != nil {
					for _, usb := range usbDetector.NewUSB {
						for _, partition := range usb.Partitions {
							for _, mountpoint := range partition.Mountpoints {
								monitor, err := monitoring.NewUSBMonitor(mountpoint, monitoring.NewFanotifyInitializer())
								if err != nil {
									log.Printf("Failed to create monitor for %s: %v\n", mountpoint, err)
									continue
								}
								go monitor.Start(context.Background())
							}
						}
					}
					usbDetector.NewUSB = nil
				}
			}
		}
	}()

	// Wait for both goroutines to finish
	<-done
	<-done // Wait for the second goroutine

	return nil
}
