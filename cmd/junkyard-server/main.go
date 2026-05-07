package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mr-andrej/junkyard/internal/api"
	"github.com/mr-andrej/junkyard/internal/ingestion"
	"github.com/mr-andrej/junkyard/internal/storage"
)

var Version = "1.0.0"

func main() {
	httpAddr := flag.String("http-addr", ":8080", "HTTP server address")
	syslogAddr := flag.String("syslog-addr", ":5514", "Syslog server address")
	dbPath := flag.String("db-path", "./junkyard.db", "SQLite database path")
	retentionDays := flag.Int("retention-days", 14, "Log retention period in days")
	version := flag.Bool("version", false, "Print version and exit")

	flag.Parse()

	if *version {
		fmt.Printf("JUNKyard v%s\n", Version)
		os.Exit(0)
	}

	os.Setenv("JUNKYARD_VERSION", Version)

	log.Printf("🗑️  Starting JUNKyard v%s", Version)
	log.Printf("📁 Database: %s", *dbPath)
	log.Printf("🌐 HTTP API: %s", *httpAddr)
	log.Printf("📡 Syslog: %s", *syslogAddr)
	log.Printf("🗓️  Retention: %d days", *retentionDays)

	// Initialize database
	db, err := storage.NewDB(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Println("✅ Database initialized")

	// Create HTTP ingestion handler
	httpHandler := ingestion.NewHTTPIngestionHandler(db)

	// Create API server
	apiServer := api.NewAPIServer(db)

	// Add ingestion routes
	apiServer.Router().HandleFunc("/api/ingest", httpHandler.HandleIngest).Methods("POST")
	apiServer.Router().HandleFunc("/api/ingest/batch", httpHandler.HandleBatchIngest).Methods("POST")

	// Start HTTP server
	httpServer := &http.Server{
		Addr:         *httpAddr,
		Handler:      apiServer,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("🚀 HTTP server started on %s", *httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Start Syslog server
	syslogServer, err := ingestion.NewSyslogServer(db, *syslogAddr)
	if err != nil {
		log.Fatalf("Failed to start syslog server: %v", err)
	}

	go func() {
		if err := syslogServer.Start(); err != nil {
			log.Printf("Syslog server error: %v", err)
		}
	}()

	// Start cleanup routine (runs once per day)
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			deleted, err := db.CleanupOldLogs(*retentionDays)
			if err != nil {
				log.Printf("Cleanup error: %v", err)
			} else if deleted > 0 {
				log.Printf("🧹 Cleaned up %d old logs", deleted)
			}
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("🛑 Shutting down JUNKyard...")
	syslogServer.Stop()
	httpServer.Close()
	log.Println("✅ JUNKyard stopped")
}
