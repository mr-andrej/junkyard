package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/mr-andrej/junkyard/internal/storage"
	"github.com/mr-andrej/junkyard/internal/web"
)

type APIServer struct {
	db        *storage.DB
	router    *mux.Router
	startTime time.Time
}

func NewAPIServer(db *storage.DB) *APIServer {
	api := &APIServer{
		db:        db,
		router:    mux.NewRouter(),
		startTime: time.Now(),
	}
	api.setupRoutes()
	return api
}

func (api *APIServer) setupRoutes() {
	// Web UI
	api.router.HandleFunc("/", web.ServeUI).Methods("GET")

	// Health check
	api.router.HandleFunc("/health", api.handleHealth).Methods("GET")

	// Log queries
	api.router.HandleFunc("/api/logs", api.handleQueryLogs).Methods("GET")
	api.router.HandleFunc("/api/hosts", api.handleHosts).Methods("GET")
	api.router.HandleFunc("/api/sources", api.handleSources).Methods("GET")

	// Statistics
	api.router.HandleFunc("/api/stats", api.handleStats).Methods("GET")
	api.router.HandleFunc("/api/timeseries", api.handleTimeSeries).Methods("GET")
}

func (api *APIServer) Router() *mux.Router {
	return api.router
}

func (api *APIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api.router.ServeHTTP(w, r)
}

// handleHealth returns server health status
func (api *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	hostname, _ := os.Hostname()
	stats, _ := api.db.GetStats()

	health := map[string]interface{}{
		"status":         "ok",
		"version":        os.Getenv("JUNKYARD_VERSION"),
		"hostname":       hostname,
		"uptime_seconds": int64(time.Since(api.startTime).Seconds()),
		"database": map[string]interface{}{
			"path":         api.db.Path(),
			"size_mb":      stats["db_size_mb"],
			"total_logs":   stats["total"],
			"last_updated": time.Now().Format(time.RFC3339),
		},
	}

	json.NewEncoder(w).Encode(health)
}

// handleQueryLogs returns logs based on filter criteria
func (api *APIServer) handleQueryLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()

	opts := storage.QueryOptions{
		Limit:   parseIntParam(query.Get("limit"), 100),
		Offset:  parseIntParam(query.Get("offset"), 0),
		Host:    query.Get("host"),
		Source:  query.Get("source"),
		Level:   query.Get("level"),
		Search:  query.Get("search"),
		OrderBy: "timestamp_desc",
	}

	if query.Get("order") == "asc" {
		opts.OrderBy = "timestamp_asc"
	}

	if start := query.Get("start"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			opts.StartTime = &t
		}
	}

	if end := query.Get("end"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			opts.EndTime = &t
		}
	}

	logs, err := api.db.Query(opts)
	if err != nil {
		http.Error(w, fmt.Sprintf("Query failed: %v", err), http.StatusInternalServerError)
		return
	}

	if logs == nil {
		logs = []storage.LogEntry{}
	}

	json.NewEncoder(w).Encode(logs)
}

// handleHosts returns list of unique hostnames
func (api *APIServer) handleHosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	hosts, err := api.db.GetHostList()
	if err != nil {
		http.Error(w, fmt.Sprintf("Query failed: %v", err), http.StatusInternalServerError)
		return
	}

	if hosts == nil {
		hosts = []string{}
	}

	json.NewEncoder(w).Encode(hosts)
}

// handleSources returns list of unique sources
func (api *APIServer) handleSources(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	sources, err := api.db.GetSourceList()
	if err != nil {
		http.Error(w, fmt.Sprintf("Query failed: %v", err), http.StatusInternalServerError)
		return
	}

	if sources == nil {
		sources = []string{}
	}

	json.NewEncoder(w).Encode(sources)
}

// handleStats returns aggregate log statistics
func (api *APIServer) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	stats, err := api.db.GetStats()
	if err != nil {
		http.Error(w, fmt.Sprintf("Stats query failed: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(stats)
}

// handleTimeSeries returns log count time-series data
func (api *APIServer) handleTimeSeries(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	interval := query.Get("interval")
	hours := parseIntParam(query.Get("hours"), 24)

	if hours > 336 { // Max 14 days
		hours = 336
	}

	data, err := api.db.GetTimeSeriesData(interval, hours)
	if err != nil {
		http.Error(w, fmt.Sprintf("Timeseries query failed: %v", err), http.StatusInternalServerError)
		return
	}

	if data == nil {
		data = []storage.TimeSeriesPoint{}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"interval": interval,
		"hours":    hours,
		"data":     data,
	})
}

// parseIntParam parses an integer query parameter with default
func parseIntParam(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	if v < 0 {
		return defaultVal
	}
	return v
}
