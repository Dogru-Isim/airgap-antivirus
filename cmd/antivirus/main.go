package main

import (
    "fmt"
    "context"
    "log"
    "os"
    "os/signal"
    "github.com/Dogru-Isim/airgap-antivirus/internal/config"
    "github.com/Dogru-Isim/airgap-antivirus/internal/monitoring"
	"syscall"
	"time"
)

func main() {
	// Create a context that cancels on interrupt signals
	ctx, stop := signal.NotifyContext(context.Background(), 
		os.Interrupt,    // ^C
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
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	log.Printf("Version: %s", cfg.Version)

	cpu, err := monitoring.NewCpu(5) // Window size
	if err != nil {
		return fmt.Errorf("cpu monitoring init failed: %w", err)
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down gracefully")
			return nil
		case <-ticker.C:
			if err := cpu.LogUsage(); err != nil {
				return fmt.Errorf("cpu monitoring error: %w", err)
			}
			log.Printf("Number of cores: %d", cpu.LogicalCores)
		}
	}
}

