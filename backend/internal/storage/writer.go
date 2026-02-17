package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/logpulse/backend/internal/models"
)

// Writer handles writing log chunks to disk
type Writer struct {
	basePath  string
	chunkSize int
	chunkSeq  int64
	mu        sync.Mutex
}

// NewWriter creates a new storage writer
func NewWriter(basePath string, chunkSize int) *Writer {
	os.MkdirAll(basePath, 0755)
	return &Writer{
		basePath:  basePath,
		chunkSize: chunkSize,
	}
}

// WriteChunk writes a batch of logs to a new chunk file
func (w *Writer) WriteChunk(labels map[string]string, entries []models.LogEntry) (string, time.Time, time.Time, error) {
	// Generate chunk ID and prepare paths outside of lock
	seq := atomic.AddInt64(&w.chunkSeq, 1)
	chunkID := fmt.Sprintf("chunk_%d_%d", time.Now().Unix(), seq)
	labelPath := models.Labels(labels).ToPath()
	dirPath := filepath.Join(w.basePath, labelPath)

	// Create directory (can be done without lock)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", time.Time{}, time.Time{}, err
	}

	// Prepare file paths
	chunkPath := filepath.Join(dirPath, chunkID+".log")
	metaPath := filepath.Join(dirPath, chunkID+".meta")

	// Calculate time range
	var startTime, endTime time.Time
	if len(entries) > 0 {
		startTime = entries[0].Timestamp
		endTime = entries[len(entries)-1].Timestamp
	}

	// Only lock for the actual file creation and writing
	w.mu.Lock()
	defer w.mu.Unlock()

	// Create and write chunk file
	file, err := os.Create(chunkPath)
	if err != nil {
		return "", time.Time{}, time.Time{}, err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, entry := range entries {
		line, _ := json.Marshal(entry)
		writer.Write(line)
		writer.WriteByte('\n')
	}

	if err := writer.Flush(); err != nil {
		return "", time.Time{}, time.Time{}, err
	}

	// Write metadata file
	meta := models.ChunkMeta{
		ID:         chunkID,
		Labels:     labels,
		StartTime:  startTime.Unix(),
		EndTime:    endTime.Unix(),
		EntryCount: len(entries),
	}

	metaFile, err := os.Create(metaPath)
	if err != nil {
		return "", time.Time{}, time.Time{}, err
	}
	defer metaFile.Close()

	json.NewEncoder(metaFile).Encode(meta)

	return chunkID, startTime, endTime, nil
}

// GetStorageSize returns total storage used in bytes
func (w *Writer) GetStorageSize() int64 {
	var size int64
	filepath.Walk(w.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

// GetChunkCount returns total number of chunks
func (w *Writer) GetChunkCount() int {
	count := 0
	filepath.Walk(w.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Ext(path) == ".log" {
			count++
		}
		return nil
	})
	return count
}
