package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type LogEntry struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Host      string    `json:"host"`
	Source    string    `json:"source"` // e.g., "syslog", "ssh", "firewall"
	Level     string    `json:"level"`  // error, warning, info, debug
	Message   string    `json:"message"`
	Raw       string    `json:"raw"` // Original log line
}

type DB struct {
	conn *sql.DB
	path string
}

// NewDB creates a new JUNKyard database connection
func NewDB(path string) (*DB, error) {
	// WAL mode for better concurrent performance
	// NORMAL synchronous mode for balance between safety and speed
	// Shared cache for multiple connections
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_synchronous=NORMAL&cache=shared", path)

	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable Write-Ahead Logging for better performance
	conn.SetMaxOpenConns(1) // SQLite performs better with single connection

	db := &DB{
		conn: conn,
		path: path,
	}

	if err := db.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return db, nil
}

func (db *DB) initialize() error {
	schema := `
    -- Main logs table
    CREATE TABLE IF NOT EXISTS logs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        host TEXT NOT NULL,
        source TEXT NOT NULL,
        level TEXT NOT NULL,
        message TEXT NOT NULL,
        raw TEXT
    );
    
    -- Indexes for fast queries
    CREATE INDEX IF NOT EXISTS idx_timestamp ON logs(timestamp DESC);
    CREATE INDEX IF NOT EXISTS idx_host ON logs(host);
    CREATE INDEX IF NOT EXISTS idx_level ON logs(level);
    CREATE INDEX IF NOT EXISTS idx_source ON logs(source);
    CREATE INDEX IF NOT EXISTS idx_host_timestamp ON logs(host, timestamp DESC);
    
    -- Full-text search using FTS5 (adds ~10MB overhead but worth it)
    CREATE VIRTUAL TABLE IF NOT EXISTS logs_fts USING fts5(
        message, 
        content=logs, 
        content_rowid=id,
        tokenize='porter unicode61'
    );
    
    -- Triggers to keep FTS in sync with main table
    CREATE TRIGGER IF NOT EXISTS logs_ai AFTER INSERT ON logs BEGIN
        INSERT INTO logs_fts(rowid, message) VALUES (new.id, new.message);
    END;
    
    CREATE TRIGGER IF NOT EXISTS logs_ad AFTER DELETE ON logs BEGIN
        DELETE FROM logs_fts WHERE rowid = old.id;
    END;
    
    -- Metadata table for JUNKyard itself
    CREATE TABLE IF NOT EXISTS metadata (
        key TEXT PRIMARY KEY,
        value TEXT,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );
    
    -- Store schema version
    INSERT OR IGNORE INTO metadata (key, value) VALUES ('schema_version', '1.0');
    INSERT OR IGNORE INTO metadata (key, value) VALUES ('created_at', datetime('now'));
    `

	_, err := db.conn.Exec(schema)
	return err
}

func (db *DB) Insert(entry *LogEntry) error {
	query := `INSERT INTO logs (timestamp, host, source, level, message, raw) 
              VALUES (?, ?, ?, ?, ?, ?)`

	_, err := db.conn.Exec(query,
		entry.Timestamp,
		entry.Host,
		entry.Source,
		entry.Level,
		entry.Message,
		entry.Raw,
	)

	if err != nil {
		return fmt.Errorf("failed to insert log: %w", err)
	}

	return nil
}

func (db *DB) InsertBatch(entries []LogEntry) (int, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO logs (timestamp, host, source, level, message, raw) 
                             VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	inserted := 0
	for _, entry := range entries {
		_, err := stmt.Exec(
			entry.Timestamp,
			entry.Host,
			entry.Source,
			entry.Level,
			entry.Message,
			entry.Raw,
		)
		if err == nil {
			inserted++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return inserted, nil
}

func (db *DB) CleanupOldLogs(retentionDays int) (int64, error) {
	query := `DELETE FROM logs WHERE timestamp < datetime('now', '-' || ? || ' days')`
	result, err := db.conn.Exec(query, retentionDays)
	if err != nil {
		return 0, fmt.Errorf("cleanup failed: %w", err)
	}

	deleted, _ := result.RowsAffected()

	// Optimize database after cleanup to reclaim space
	if deleted > 0 {
		db.conn.Exec("VACUUM")
	}

	return deleted, nil
}

func (db *DB) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total logs
	var total int64
	db.conn.QueryRow("SELECT COUNT(*) FROM logs").Scan(&total)
	stats["total"] = total

	// Logs by level
	rows, _ := db.conn.Query(`
        SELECT level, COUNT(*) as count 
        FROM logs 
        GROUP BY level
    `)
	defer rows.Close()

	levelCounts := make(map[string]int64)
	for rows.Next() {
		var level string
		var count int64
		rows.Scan(&level, &count)
		levelCounts[level] = count
	}
	stats["by_level"] = levelCounts

	// Top hosts by log volume
	rows, _ = db.conn.Query(`
        SELECT host, COUNT(*) as count 
        FROM logs 
        GROUP BY host 
        ORDER BY count DESC 
        LIMIT 10
    `)
	defer rows.Close()

	hostCounts := make(map[string]int64)
	for rows.Next() {
		var host string
		var count int64
		rows.Scan(&host, &count)
		hostCounts[host] = count
	}
	stats["by_host"] = hostCounts

	// Top sources
	rows, _ = db.conn.Query(`
        SELECT source, COUNT(*) as count 
        FROM logs 
        GROUP BY source 
        ORDER BY count DESC
    `)
	defer rows.Close()

	sourceCounts := make(map[string]int64)
	for rows.Next() {
		var source string
		var count int64
		rows.Scan(&source, &count)
		sourceCounts[source] = count
	}
	stats["by_source"] = sourceCounts

	// Database file size
	var pageCount, pageSize int64
	db.conn.QueryRow("PRAGMA page_count").Scan(&pageCount)
	db.conn.QueryRow("PRAGMA page_size").Scan(&pageSize)
	stats["db_size_mb"] = float64(pageCount*pageSize) / 1024 / 1024

	// Oldest and newest log
	var oldest, newest string
	db.conn.QueryRow("SELECT MIN(timestamp), MAX(timestamp) FROM logs").Scan(&oldest, &newest)
	stats["oldest_log"] = oldest
	stats["newest_log"] = newest

	// Logs in last hour
	var lastHour int64
	db.conn.QueryRow("SELECT COUNT(*) FROM logs WHERE timestamp >= datetime('now', '-1 hour')").Scan(&lastHour)
	stats["last_hour"] = lastHour

	// Logs in last 24 hours
	var last24h int64
	db.conn.QueryRow("SELECT COUNT(*) FROM logs WHERE timestamp >= datetime('now', '-1 day')").Scan(&last24h)
	stats["last_24h"] = last24h

	return stats, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Path() string {
	return db.path
}
