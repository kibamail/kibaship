package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kibamail/kibaship/internal/registryauth"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("starting registry auth service...")

	// Load configuration
	config := registryauth.LoadConfig()

	// Create server
	server, err := registryauth.NewServer(config)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	// Start server in background
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	log.Println("received shutdown signal")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("error during shutdown: %v", err)
	}

	log.Println("registry auth service stopped")
}
