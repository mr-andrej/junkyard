package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/mr-andrej/junkyard/internal/api"
	"github.com/mr-andrej/junkyard/internal/ingestion"
	"github.com/mr-andrej/junkyard/internal/model"
	"github.com/mr-andrej/junkyard/internal/otlp"
	"github.com/mr-andrej/junkyard/internal/pipeline"
	"github.com/mr-andrej/junkyard/internal/storage"
	logstore "github.com/mr-andrej/junkyard/internal/storage/logs"
)

var Version = "1.0.0"

func main() {
	httpAddr := flag.String("http-addr", ":8080", "HTTP server address")
	syslogAddr := flag.String("syslog-addr", ":5514", "Syslog server address")
	dbPath := flag.String("db-path", "./junkyard.db", "SQLite database path")
	v2DBPath := flag.String("v2-db-path", "./junkyard-v2.db", "v2 SQLite log store path")
	otlpGRPCAddr := flag.String("otlp-grpc-addr", ":4317", "OTLP gRPC receiver address")
	otlpHTTPAddr := flag.String("otlp-http-addr", ":4318", "OTLP HTTP receiver address")
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

	// v2: log store, ingest pipeline, OTLP receivers
	logStore, err := logstore.Open(*v2DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize v2 log store: %v", err)
	}

	pipe := pipeline.New(4096, 500, time.Second, logStore.WriteBatch)

	// Create HTTP ingestion handler
	httpHandler := ingestion.NewHTTPIngestionHandler(db)

	// Create API server
	apiServer := api.NewAPIServer(db)

	// Add ingestion routes
	apiServer.Router().HandleFunc("/api/ingest", httpHandler.HandleIngest).Methods("POST")
	apiServer.Router().HandleFunc("/api/ingest/batch", httpHandler.HandleBatchIngest).Methods("POST")

	// v2 query API
	apiServer.Router().HandleFunc("/api/v1/logs", handleV2Logs(logStore)).Methods("GET")

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

	// Start OTLP receivers (v2)
	otlpGRPCServer := otlp.NewGRPCServer(pipe)
	grpcListener, err := net.Listen("tcp", *otlpGRPCAddr)
	if err != nil {
		log.Fatalf("Failed to start OTLP gRPC listener: %v", err)
	}

	go func() {
		log.Printf("🚀 OTLP gRPC receiver started on %s", *otlpGRPCAddr)
		if err := otlpGRPCServer.Serve(grpcListener); err != nil {
			log.Printf("OTLP gRPC server error: %v", err)
		}
	}()

	otlpHTTPServer := &http.Server{
		Addr:         *otlpHTTPAddr,
		Handler:      otlp.NewHTTPHandler(pipe),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("🚀 OTLP HTTP receiver started on %s", *otlpHTTPAddr)
		if err := otlpHTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("OTLP HTTP server error: %v", err)
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
	otlpGRPCServer.GracefulStop()
	otlpHTTPServer.Close()
	pipe.Shutdown() // drains the queue and flushes the final batch
	logStore.Close()
	syslogServer.Stop()
	httpServer.Close()
	log.Println("✅ JUNKyard stopped")
}

// handleV2Logs serves GET /api/v1/logs from the v2 log store.
func handleV2Logs(store *logstore.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		query := r.URL.Query()

		opts := logstore.QueryOptions{
			Service: query.Get("service"),
			Query:   query.Get("q"),
			Limit:   100,
		}

		if severity := query.Get("severity"); severity != "" {
			if n, err := strconv.Atoi(severity); err == nil && n > 0 {
				opts.SeverityMin = int32(n)
			}
		}

		if from := query.Get("from"); from != "" {
			if t, err := time.Parse(time.RFC3339, from); err == nil {
				opts.From = t
			}
		}

		if to := query.Get("to"); to != "" {
			if t, err := time.Parse(time.RFC3339, to); err == nil {
				opts.To = t
			}
		}

		if limit := query.Get("limit"); limit != "" {
			if n, err := strconv.Atoi(limit); err == nil && n > 0 {
				opts.Limit = n
			}
		}

		entries, err := store.Query(opts)
		if err != nil {
			http.Error(w, fmt.Sprintf("Query failed: %v", err), http.StatusInternalServerError)
			return
		}

		if entries == nil {
			entries = []model.LogEntry{}
		}

		json.NewEncoder(w).Encode(entries)
	}
}
