package models

import "time"

// Chunk represents a chunk of logs for a specific label set
type Chunk struct {
	ID         string            `json:"id"`
	Labels     map[string]string `json:"labels"`
	StartTime  time.Time         `json:"startTime"`
	EndTime    time.Time         `json:"endTime"`
	Size       int64             `json:"size"`
	EntryCount int               `json:"entryCount"`
	FilePath   string            `json:"filePath"`
}

// ChunkMeta is stored alongside chunk data for quick lookups
type ChunkMeta struct {
	ID         string            `json:"id"`
	Labels     map[string]string `json:"labels"`
	StartTime  int64             `json:"start_time"` // Unix timestamp
	EndTime    int64             `json:"end_time"`
	EntryCount int               `json:"entry_count"`
}
