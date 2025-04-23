package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"github.com/HiteshManglani123/air-gapped/software-based-detection/antivirus-project/internal/config/loader.go"
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
	cfg, err := config.load()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	log.Printf("Starting server v%s in %s mode on port %d",
		cfg.Version)

	// Add your server setup here

	<-ctx.Done()
	log.Println("Shutting down gracefully")
	return nil
}

