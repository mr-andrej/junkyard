package logs

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mr-andrej/junkyard/internal/model"
)

func openTempStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func seedEntries() []model.LogEntry {
	base := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	return []model.LogEntry{
		{
			Timestamp:      base,
			SeverityNumber: 9,
			SeverityText:   "INFO",
			Body:           "user login successful",
			Service:        "auth",
			Attributes:     map[string]string{"env": "prod"},
			TraceID:        "deadbeef000102030405060708090a0b",
			SpanID:         "0102030405060708",
		},
		{
			Timestamp:      base.Add(time.Minute),
			SeverityNumber: 13,
			SeverityText:   "WARN",
			Body:           "disk usage above threshold",
			Service:        "monitor",
		},
		{
			Timestamp:      base.Add(2 * time.Minute),
			SeverityNumber: 17,
			SeverityText:   "ERROR",
			Body:           "login failed for user admin",
			Service:        "auth",
		},
	}
}

func writeSeed(t *testing.T, store *Store) {
	t.Helper()
	if err := store.WriteBatch(seedEntries()); err != nil {
		t.Fatalf("write batch: %v", err)
	}
}

func TestWriteBatchAndReadBack(t *testing.T) {
	store := openTempStore(t)
	writeSeed(t, store)

	entries, err := store.Query(QueryOptions{Limit: 10})
	if err != nil {
		t.Fatalf("query: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// Newest first
	if entries[0].Body != "login failed for user admin" {
		t.Fatalf("expected newest entry first, got %q", entries[0].Body)
	}

	// Full field round-trip on the oldest entry
	oldest := entries[2]
	if oldest.Service != "auth" || oldest.SeverityText != "INFO" || oldest.SeverityNumber != 9 {
		t.Errorf("unexpected oldest entry: %+v", oldest)
	}
	if oldest.TraceID != "deadbeef000102030405060708090a0b" || oldest.SpanID != "0102030405060708" {
		t.Errorf("trace/span ID mismatch: %q / %q", oldest.TraceID, oldest.SpanID)
	}
	if oldest.Attributes["env"] != "prod" {
		t.Errorf("attribute env: got %q", oldest.Attributes["env"])
	}
	if want := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC); !oldest.Timestamp.Equal(want) {
		t.Errorf("timestamp: expected %v, got %v", want, oldest.Timestamp)
	}
}

func TestQueryFilters(t *testing.T) {
	from := time.Date(2026, 7, 20, 12, 0, 30, 0, time.UTC)
	to := time.Date(2026, 7, 20, 12, 2, 0, 0, time.UTC)

	tests := []struct {
		name       string
		opts       QueryOptions
		wantBodies []string
	}{
		{
			name:       "no filter",
			opts:       QueryOptions{Limit: 10},
			wantBodies: []string{"login failed for user admin", "disk usage above threshold", "user login successful"},
		},
		{
			name:       "by service",
			opts:       QueryOptions{Service: "auth", Limit: 10},
			wantBodies: []string{"login failed for user admin", "user login successful"},
		},
		{
			name:       "by minimum severity",
			opts:       QueryOptions{SeverityMin: 13, Limit: 10},
			wantBodies: []string{"login failed for user admin", "disk usage above threshold"},
		},
		{
			name:       "by full-text query",
			opts:       QueryOptions{Query: "login", Limit: 10},
			wantBodies: []string{"login failed for user admin", "user login successful"},
		},
		{
			name:       "by time range",
			opts:       QueryOptions{From: from, To: to, Limit: 10},
			wantBodies: []string{"login failed for user admin", "disk usage above threshold"},
		},
		{
			name:       "combined filters",
			opts:       QueryOptions{Service: "auth", SeverityMin: 13, Query: "login", Limit: 10},
			wantBodies: []string{"login failed for user admin"},
		},
		{
			name:       "limit",
			opts:       QueryOptions{Limit: 2},
			wantBodies: []string{"login failed for user admin", "disk usage above threshold"},
		},
		{
			name:       "no matches",
			opts:       QueryOptions{Service: "nonexistent", Limit: 10},
			wantBodies: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := openTempStore(t)
			writeSeed(t, store)

			entries, err := store.Query(tt.opts)
			if err != nil {
				t.Fatalf("query: %v", err)
			}

			if len(entries) != len(tt.wantBodies) {
				t.Fatalf("expected %d entries, got %d (%+v)", len(tt.wantBodies), len(entries), entries)
			}
			for i, want := range tt.wantBodies {
				if entries[i].Body != want {
					t.Errorf("entry %d: expected %q, got %q", i, want, entries[i].Body)
				}
			}
		})
	}
}

func TestLikeFallbackWhenFTS5Unavailable(t *testing.T) {
	store := openTempStore(t)
	store.hasFTS5 = false
	writeSeed(t, store)

	entries, err := store.Query(QueryOptions{Query: "threshold", Limit: 10})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(entries) != 1 || entries[0].Body != "disk usage above threshold" {
		t.Fatalf("expected the disk-usage entry, got %+v", entries)
	}
}

func TestWriteEmptyBatch(t *testing.T) {
	store := openTempStore(t)
	if err := store.WriteBatch(nil); err != nil {
		t.Fatalf("write empty batch: %v", err)
	}
}
