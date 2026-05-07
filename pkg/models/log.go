package models

import "time"

// LogEntry represents a single log entry in JUNKyard
type LogEntry struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Host      string    `json:"host"`
	Source    string    `json:"source"` // e.g., "syslog", "ssh", "firewall", "http"
	Level     string    `json:"level"`  // error, warning, info, debug
	Message   string    `json:"message"`
	Raw       string    `json:"raw"` // Original log line
}

// LogLevel represents severity levels for logs
type LogLevel string

const (
	Debug   LogLevel = "debug"
	Info    LogLevel = "info"
	Warning LogLevel = "warning"
	Error   LogLevel = "error"
)

// IngestRequest is used for HTTP API log ingestion
type IngestRequest struct {
	Timestamp *time.Time `json:"timestamp,omitempty"`
	Host      string     `json:"host"`
	Source    string     `json:"source,omitempty"`
	Level     string     `json:"level,omitempty"`
	Message   string     `json:"message"`
	Raw       string     `json:"raw,omitempty"`
}

// IngestResponse is returned after successful ingestion
type IngestResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// BatchIngestResponse is returned after batch ingestion
type BatchIngestResponse struct {
	Status   string `json:"status"`
	Inserted int    `json:"inserted"`
	Total    int    `json:"total"`
	Message  string `json:"message"`
}

// QueryRequest represents parameters for querying logs
type QueryRequest struct {
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
	Host   string `json:"host"`
	Source string `json:"source"`
	Level  string `json:"level"`
	Search string `json:"search"`
}

// QueryResponse is returned from log queries
type QueryResponse struct {
	Total   int64      `json:"total"`
	Limit   int        `json:"limit"`
	Offset  int        `json:"offset"`
	Results []LogEntry `json:"results"`
}

// StatsResponse contains aggregate log statistics
type StatsResponse struct {
	Total     int64            `json:"total"`
	ByLevel   map[string]int64 `json:"by_level"`
	ByHost    map[string]int64 `json:"by_host"`
	BySource  map[string]int64 `json:"by_source"`
	DBSizeMB  float64          `json:"db_size_mb"`
	OldestLog string           `json:"oldest_log"`
	NewestLog string           `json:"newest_log"`
	LastHour  int64            `json:"last_hour"`
	Last24h   int64            `json:"last_24h"`
}

// TimeSeriesPoint represents a point in time-series data
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Count     int64     `json:"count"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status  string         `json:"status"`
	Version string         `json:"version"`
	Uptime  int64          `json:"uptime_seconds"`
	DB      DatabaseHealth `json:"database"`
}

// DatabaseHealth represents database status
type DatabaseHealth struct {
	Path        string  `json:"path"`
	SizeMB      float64 `json:"size_mb"`
	TotalLogs   int64   `json:"total_logs"`
	LastUpdated string  `json:"last_updated"`
}
