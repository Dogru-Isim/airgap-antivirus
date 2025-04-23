package main

import (
    "fmt"
    "context"
    "log"
    "os"
    "os/signal"
    "github.com/Dogru-Isim/airgap-antivirus/internal/config"
)

func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
    defer cancel()

    if err := run(ctx); err != nil {
        log.Fatalf("Fatal error: %v", err)
    }
}

// run contains the main application logic
func run(ctx context.Context) error {
    cfg, err := config.Load()
    if err != nil {
        return fmt.Errorf("config error: %w", err)
    }

    log.Printf("Version: %s", cfg.Version)

    <-ctx.Done()
    log.Println("Shutting down gracefully")
    return nil
}

