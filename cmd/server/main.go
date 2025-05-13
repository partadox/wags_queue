package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/partadox/wags_queue/internal/api"
	"github.com/partadox/wags_queue/internal/config"
	"github.com/partadox/wags_queue/internal/db"
	"github.com/partadox/wags_queue/internal/worker"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connection
	database, err := db.Connect(cfg.DB)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize worker
	msgWorker := worker.NewMessageWorker(database, cfg.ExternalAPI)
	go msgWorker.Run()

	// Initialize bulk message processor
	bulkProcessor := worker.NewBulkProcessor(database)
	go bulkProcessor.Run()

	// Start the API server
	apiServer := api.NewServer(cfg, database)
	go func() {
		log.Printf("Starting server on port %s...", cfg.Server.Port)
		if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for termination signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	
	// Allow up to 10 seconds for graceful shutdown
	// You might want to adjust this depending on your requirements
	shutdownTimeout := 10 * time.Second
	
	// Stop workers first
	msgWorker.Stop()
	bulkProcessor.Stop()
	
	// Then stop the API server
	if err := apiServer.Stop(shutdownTimeout); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
