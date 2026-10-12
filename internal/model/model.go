package model

import "time"

// LogEntry is the normalized internal representation of a single log record.
// Receivers convert OTLP protobuf types into LogEntry at the pipeline edge;
// the storage layer never sees OTLP types.
type LogEntry struct {
	Timestamp      time.Time         `json:"timestamp"`
	SeverityNumber int32             `json:"severity_number"`
	SeverityText   string            `json:"severity_text"`
	Body           string            `json:"body"`
	Service        string            `json:"service"` // from the resource's service.name attribute
	Attributes     map[string]string `json:"attributes,omitempty"`
	TraceID        string            `json:"trace_id,omitempty"` // hex, empty when absent
	SpanID         string            `json:"span_id,omitempty"`  // hex, empty when absent
}
