package ingestion

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mr-andrej/junkyard/internal/storage"
)

type HTTPIngestionHandler struct {
	db *storage.DB
}

func NewHTTPIngestionHandler(db *storage.DB) *HTTPIngestionHandler {
	return &HTTPIngestionHandler{db: db}
}

// HandleIngest handles single log ingestion via HTTP POST
func (h *HTTPIngestionHandler) HandleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var entry storage.LogEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Set timestamp if not provided
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Validate required fields
	if entry.Host == "" {
		http.Error(w, "Missing required field: host", http.StatusBadRequest)
		return
	}
	if entry.Message == "" {
		http.Error(w, "Missing required field: message", http.StatusBadRequest)
		return
	}

	// Set defaults
	if entry.Level == "" {
		entry.Level = "info"
	}
	if entry.Source == "" {
		entry.Source = "http"
	}

	// Store raw JSON if not provided
	if entry.Raw == "" {
		rawBytes, _ := json.Marshal(entry)
		entry.Raw = string(rawBytes)
	}

	// Insert into JUNKyard
	if err := h.db.Insert(&entry); err != nil {
		http.Error(w, fmt.Sprintf("Failed to store log: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Log thrown into the junkyard",
	})
}

// HandleBatchIngest handles multiple logs in a single request
func (h *HTTPIngestionHandler) HandleBatchIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var entries []storage.LogEntry
	if err := json.NewDecoder(r.Body).Decode(&entries); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if len(entries) == 0 {
		http.Error(w, "Empty batch", http.StatusBadRequest)
		return
	}

	if len(entries) > 1000 {
		http.Error(w, "Batch too large (max 1000 logs)", http.StatusBadRequest)
		return
	}

	// Prepare entries
	now := time.Now()
	for i := range entries {
		if entries[i].Timestamp.IsZero() {
			entries[i].Timestamp = now
		}
		if entries[i].Level == "" {
			entries[i].Level = "info"
		}
		if entries[i].Source == "" {
			entries[i].Source = "http"
		}
		if entries[i].Raw == "" {
			rawBytes, _ := json.Marshal(entries[i])
			entries[i].Raw = string(rawBytes)
		}
	}

	// Batch insert
	inserted, err := h.db.InsertBatch(entries)
	if err != nil {
		http.Error(w, fmt.Sprintf("Batch insert failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"inserted": inserted,
		"total":    len(entries),
		"message":  fmt.Sprintf("Dumped %d logs into the junkyard", inserted),
	})
}
