package models

import "time"

// LogEntry represents a single log line with metadata
type LogEntry struct {
	ID        string            `json:"id"`
	Timestamp time.Time         `json:"timestamp"`
	Line      string            `json:"message"`
	Labels    map[string]string `json:"labels"`
}

// IngestRequest is the incoming log payload
type IngestRequest struct {
	Streams []Stream `json:"streams"`
}

type Stream struct {
	Labels  map[string]string `json:"labels"`
	Entries []Entry           `json:"entries"`
}

type Entry struct {
	Ts   string `json:"ts"`
	Line string `json:"line"`
}

// IngestResponse confirms ingestion
type IngestResponse struct {
	Accepted int `json:"accepted"`
}
