package storage

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

// StartRetentionWorker starts a background worker to clean up old logs
func StartRetentionWorker(basePath string, retentionDays int) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		CleanupOldChunks(basePath, retentionDays)
	}
}

// CleanupOldChunks removes chunk files older than retention period
func CleanupOldChunks(basePath string, retentionDays int) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	deletedCount := 0
	deletedBytes := int64(0)

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file is older than cutoff
		if info.ModTime().Before(cutoff) {
			size := info.Size()
			if err := os.Remove(path); err != nil {
				log.Printf("Failed to delete %s: %v", path, err)
				return nil
			}
			deletedCount++
			deletedBytes += size
		}

		return nil
	})

	if err != nil {
		log.Printf("Retention cleanup error: %v", err)
	}

	if deletedCount > 0 {
		log.Printf("Retention cleanup: deleted %d files (%d bytes)", deletedCount, deletedBytes)
	}

	// Remove empty directories
	cleanupEmptyDirs(basePath)
}

// cleanupEmptyDirs removes empty directories
func cleanupEmptyDirs(basePath string) {
	filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() || path == basePath {
			return nil
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return nil
		}

		if len(entries) == 0 {
			os.Remove(path)
		}

		return nil
	})
}
