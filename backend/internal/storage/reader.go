package storage

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/logpulse/backend/internal/models"
)

// Reader handles reading log chunks from disk
type Reader struct {
	basePath string
}

// NewReader creates a new storage reader
func NewReader(basePath string) *Reader {
	return &Reader{basePath: basePath}
}

// ReadChunk reads all entries from a chunk file
func (r *Reader) ReadChunk(labels map[string]string, chunkID string) ([]models.LogEntry, error) {
	labelPath := models.Labels(labels).ToPath()
	chunkPath := filepath.Join(r.basePath, labelPath, chunkID+".log")

	file, err := os.Open(chunkPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []models.LogEntry
	scanner := bufio.NewScanner(file)
	
	// Increase buffer size for large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		var entry models.LogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	return entries, scanner.Err()
}

// ReadChunkFiltered reads entries from a chunk with time filtering
func (r *Reader) ReadChunkFiltered(labels map[string]string, chunkID string, startTime, endTime time.Time) ([]models.LogEntry, int, error) {
	entries, err := r.ReadChunk(labels, chunkID)
	if err != nil {
		return nil, 0, err
	}

	scannedLines := len(entries)
	filtered := make([]models.LogEntry, 0)

	for _, entry := range entries {
		if entry.Timestamp.Before(startTime) || entry.Timestamp.After(endTime) {
			continue
		}
		filtered = append(filtered, entry)
	}

	return filtered, scannedLines, nil
}

// ReadChunkFilteredWithPagination reads entries with pagination support (limit + offset)
func (r *Reader) ReadChunkFilteredWithPagination(labels map[string]string, chunkID string, startTime, endTime time.Time, limit, offset int) ([]models.LogEntry, int, error) {
	entries, err := r.ReadChunk(labels, chunkID)
	if err != nil {
		return nil, 0, err
	}

	scannedLines := len(entries)
	filtered := make([]models.LogEntry, 0, limit) // Preallocate

	count := 0
	skipped := 0

	for _, entry := range entries {
		// Time range filter
		if entry.Timestamp.Before(startTime) || entry.Timestamp.After(endTime) {
			continue
		}

		// Skip first 'offset' entries
		if skipped < offset {
			skipped++
			continue
		}

		// Stop when we have enough
		if limit > 0 && count >= limit {
			break
		}

		filtered = append(filtered, entry)
		count++
	}

	return filtered, scannedLines, nil
}

// GetChunkMeta reads chunk metadata
func (r *Reader) GetChunkMeta(labels map[string]string, chunkID string) (*models.ChunkMeta, error) {
	labelPath := models.Labels(labels).ToPath()
	metaPath := filepath.Join(r.basePath, labelPath, chunkID+".meta")

	file, err := os.Open(metaPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var meta models.ChunkMeta
	if err := json.NewDecoder(file).Decode(&meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

// ListChunks returns all chunk IDs for a label set
func (r *Reader) ListChunks(labels map[string]string) ([]string, error) {
	labelPath := models.Labels(labels).ToPath()
	dirPath := filepath.Join(r.basePath, labelPath)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	chunks := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".log" {
			chunkID := entry.Name()[:len(entry.Name())-4] // Remove .log extension
			chunks = append(chunks, chunkID)
		}
	}

	return chunks, nil
}
