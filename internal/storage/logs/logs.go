package logs

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/mr-andrej/junkyard/internal/model"
)

// Store is the v2 SQLite log store. It lives in its own database file
// (junkyard-v2.db by default) so it never collides with v1's schema.
type Store struct {
	conn    *sql.DB
	path    string
	hasFTS5 bool
}

// QueryOptions filters log queries. Zero values mean "no filter".
type QueryOptions struct {
	Service     string
	SeverityMin int32
	Query       string // full-text match (FTS5 when available, LIKE otherwise)
	From        time.Time
	To          time.Time
	Limit       int
}

// Open creates or opens the v2 log store at path.
func Open(path string) (*Store, error) {
	// WAL mode for concurrent readers during writes, NORMAL synchronous for
	// a balance between safety and speed.
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_synchronous=NORMAL&cache=shared", path)

	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	conn.SetMaxOpenConns(1) // SQLite performs better with single connection

	store := &Store{conn: conn, path: path}
	if err := store.initialize(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return store, nil
}

func (s *Store) initialize() error {
	coreSchema := `
    CREATE TABLE IF NOT EXISTS logs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        timestamp INTEGER NOT NULL,           -- unix nanoseconds
        severity_number INTEGER NOT NULL DEFAULT 0,
        severity_text TEXT NOT NULL DEFAULT '',
        body TEXT NOT NULL,
        service TEXT NOT NULL DEFAULT '',
        attributes TEXT NOT NULL DEFAULT '{}', -- JSON object
        trace_id TEXT NOT NULL DEFAULT '',
        span_id TEXT NOT NULL DEFAULT ''
    );

    CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp DESC);
    CREATE INDEX IF NOT EXISTS idx_logs_service_timestamp ON logs(service, timestamp DESC);
    `

	if _, err := s.conn.Exec(coreSchema); err != nil {
		return err
	}

	// Optional FTS5 schema - attempt but don't fail if unavailable
	ftsSchema := `
    CREATE VIRTUAL TABLE IF NOT EXISTS logs_fts USING fts5(
        body,
        content=logs,
        content_rowid=id,
        tokenize='porter unicode61'
    );

    CREATE TRIGGER IF NOT EXISTS logs_ai AFTER INSERT ON logs BEGIN
        INSERT INTO logs_fts(rowid, body) VALUES (new.id, new.body);
    END;

    CREATE TRIGGER IF NOT EXISTS logs_ad AFTER DELETE ON logs BEGIN
        DELETE FROM logs_fts WHERE rowid = old.id;
    END;
    `

	// Try to enable FTS5, but continue without it if unavailable
	if _, err := s.conn.Exec(ftsSchema); err != nil {
		fmt.Printf("⚠️  FTS5 not available (using LIKE search instead): %v\n", err)
		s.hasFTS5 = false
	} else {
		s.hasFTS5 = true
	}

	return nil
}

// WriteBatch inserts one batch of entries in a single transaction. It is
// called from the pipeline's single writer goroutine.
func (s *Store) WriteBatch(entries []model.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	tx, err := s.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO logs
	    (timestamp, severity_number, severity_text, body, service, attributes, trace_id, span_id)
	    VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, entry := range entries {
		attrs, err := json.Marshal(entry.Attributes)
		if err != nil {
			return fmt.Errorf("failed to marshal attributes: %w", err)
		}

		if _, err := stmt.Exec(
			entry.Timestamp.UnixNano(),
			entry.SeverityNumber,
			entry.SeverityText,
			entry.Body,
			entry.Service,
			string(attrs),
			entry.TraceID,
			entry.SpanID,
		); err != nil {
			return fmt.Errorf("failed to insert log: %w", err)
		}
	}

	return tx.Commit()
}

// Query returns logs matching the filters, newest first.
func (s *Store) Query(opts QueryOptions) ([]model.LogEntry, error) {
	if opts.Limit <= 0 {
		opts.Limit = 100
	}

	var where []string
	var args []interface{}

	if opts.Service != "" {
		where = append(where, "service = ?")
		args = append(args, opts.Service)
	}
	if opts.SeverityMin > 0 {
		where = append(where, "severity_number >= ?")
		args = append(args, opts.SeverityMin)
	}
	if !opts.From.IsZero() {
		where = append(where, "timestamp >= ?")
		args = append(args, opts.From.UnixNano())
	}
	if !opts.To.IsZero() {
		where = append(where, "timestamp <= ?")
		args = append(args, opts.To.UnixNano())
	}

	query := `SELECT l.timestamp, l.severity_number, l.severity_text, l.body,
	                 l.service, l.attributes, l.trace_id, l.span_id
	          FROM logs l`

	if opts.Query != "" {
		if s.hasFTS5 {
			query += ` JOIN logs_fts f ON f.rowid = l.id`
			where = append(where, "f MATCH ?")
			args = append(args, opts.Query)
		} else {
			where = append(where, "l.body LIKE ?")
			args = append(args, "%"+opts.Query+"%")
		}
	}

	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY l.timestamp DESC LIMIT ?"
	args = append(args, opts.Limit)

	rows, err := s.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	var entries []model.LogEntry
	for rows.Next() {
		var e model.LogEntry
		var ts int64
		var attrs string

		if err := rows.Scan(&ts, &e.SeverityNumber, &e.SeverityText, &e.Body,
			&e.Service, &attrs, &e.TraceID, &e.SpanID); err != nil {
			return nil, fmt.Errorf("failed to scan log: %w", err)
		}

		e.Timestamp = time.Unix(0, ts).UTC()
		if attrs != "" && attrs != "null" {
			if err := json.Unmarshal([]byte(attrs), &e.Attributes); err != nil {
				return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
			}
		}

		entries = append(entries, e)
	}

	return entries, rows.Err()
}

// HasFTS5 reports whether full-text search is available.
func (s *Store) HasFTS5() bool {
	return s.hasFTS5
}

// Path returns the database file path.
func (s *Store) Path() string {
	return s.path
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.conn.Close()
}
