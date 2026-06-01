package storage

import (
	"fmt"
	"strings"
	"time"
)

type QueryOptions struct {
	Limit     int
	Offset    int
	Host      string
	Source    string
	Level     string
	Search    string // Full-text search
	StartTime *time.Time
	EndTime   *time.Time
	OrderBy   string // "timestamp_desc" (default), "timestamp_asc"
}

func (db *DB) Query(opts QueryOptions) ([]LogEntry, error) {
	var conditions []string
	var args []interface{}

	// Build WHERE clause
	if opts.Host != "" {
		conditions = append(conditions, "host = ?")
		args = append(args, opts.Host)
	}

	if opts.Source != "" {
		conditions = append(conditions, "source = ?")
		args = append(args, opts.Source)
	}

	if opts.Level != "" {
		conditions = append(conditions, "level = ?")
		args = append(args, opts.Level)
	}

	if opts.Search != "" {
		// Use FTS5 for full-text search if available, otherwise fall back to LIKE
		if db.hasFTS5 {
			conditions = append(conditions, "id IN (SELECT rowid FROM logs_fts WHERE message MATCH ?)")
		} else {
			conditions = append(conditions, "message LIKE ?")
			opts.Search = "%" + opts.Search + "%"
		}
		args = append(args, opts.Search)
	}

	if opts.StartTime != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, opts.StartTime)
	}

	if opts.EndTime != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, opts.EndTime)
	}

	whereClause := "1=1"
	if len(conditions) > 0 {
		whereClause = strings.Join(conditions, " AND ")
	}

	// Order by
	orderBy := "timestamp DESC"
	if opts.OrderBy == "timestamp_asc" {
		orderBy = "timestamp ASC"
	}

	// Limit and offset
	limit := opts.Limit
	if limit == 0 {
		limit = 100 // Default
	}
	if limit > 10000 {
		limit = 10000 // Max limit to prevent memory issues
	}

	query := fmt.Sprintf(`
        SELECT id, timestamp, host, source, level, message, raw 
        FROM logs 
        WHERE %s 
        ORDER BY %s 
        LIMIT ? OFFSET ?
    `, whereClause, orderBy)

	args = append(args, limit, opts.Offset)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var entries []LogEntry
	for rows.Next() {
		var entry LogEntry
		err := rows.Scan(
			&entry.ID,
			&entry.Timestamp,
			&entry.Host,
			&entry.Source,
			&entry.Level,
			&entry.Message,
			&entry.Raw,
		)
		if err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Count     int64     `json:"count"`
}

func (db *DB) GetTimeSeriesData(interval string, hours int) ([]TimeSeriesPoint, error) {
	// Returns log counts grouped by time interval
	// interval: "hour", "minute", "5min", "15min"

	var intervalSQL string
	switch interval {
	case "minute":
		intervalSQL = "strftime('%Y-%m-%d %H:%M:00', timestamp)"
	case "5min":
		intervalSQL = "strftime('%Y-%m-%d %H:', timestamp) || printf('%02d', CAST(strftime('%M', timestamp) AS INTEGER) / 5 * 5) || ':00'"
	case "15min":
		intervalSQL = "strftime('%Y-%m-%d %H:', timestamp) || printf('%02d', CAST(strftime('%M', timestamp) AS INTEGER) / 15 * 15) || ':00'"
	default: // hour
		intervalSQL = "strftime('%Y-%m-%d %H:00:00', timestamp)"
	}

	query := fmt.Sprintf(`
        SELECT %s as interval, COUNT(*) as count
        FROM logs
        WHERE timestamp >= datetime('now', '-' || ? || ' hours')
        GROUP BY interval
        ORDER BY interval ASC
    `, intervalSQL)

	rows, err := db.conn.Query(query, hours)
	if err != nil {
		return nil, fmt.Errorf("timeseries query failed: %w", err)
	}
	defer rows.Close()

	var points []TimeSeriesPoint
	for rows.Next() {
		var ts string
		var count int64
		rows.Scan(&ts, &count)

		t, _ := time.Parse("2006-01-02 15:04:05", ts)
		points = append(points, TimeSeriesPoint{
			Timestamp: t,
			Count:     count,
		})
	}

	return points, nil
}

func (db *DB) GetRecentErrors(limit int) ([]LogEntry, error) {
	return db.Query(QueryOptions{
		Level:   "error",
		Limit:   limit,
		OrderBy: "timestamp_desc",
	})
}

func (db *DB) GetHostList() ([]string, error) {
	rows, err := db.conn.Query("SELECT DISTINCT host FROM logs ORDER BY host")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []string
	for rows.Next() {
		var host string
		rows.Scan(&host)
		hosts = append(hosts, host)
	}

	return hosts, nil
}

func (db *DB) GetSourceList() ([]string, error) {
	rows, err := db.conn.Query("SELECT DISTINCT source FROM logs ORDER BY source")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []string
	for rows.Next() {
		var source string
		rows.Scan(&source)
		sources = append(sources, source)
	}

	return sources, nil
}
