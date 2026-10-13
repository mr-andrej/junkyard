package model

import (
	"strings"
	"time"
)

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

// severityNumbers maps OTLP severity text to the base of its severity-number
// range, per the OTLP spec.
var severityNumbers = map[string]int32{
	"TRACE": 1, "DEBUG": 5, "INFO": 9,
	"WARN": 13, "WARNING": 13,
	"ERROR": 17, "FATAL": 21,
}

// SeverityNumberFromText maps a severity text ("WARN") to its OTLP severity
// number. Returns 0 (unspecified) for unknown or empty values.
func SeverityNumberFromText(text string) int32 {
	return severityNumbers[strings.ToUpper(strings.TrimSpace(text))]
}
