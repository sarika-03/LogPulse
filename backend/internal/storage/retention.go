package storage

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"
)

// StartRetentionWorker starts a background worker to clean up old logs with context support
func StartRetentionWorker(ctx context.Context, basePath string, retentionDays int) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	log.Printf("[RetentionWorker] Starting with %d days retention", retentionDays)

	for {
		select {
		case <-ctx.Done():
			log.Println("[RetentionWorker] Shutting down")
			return
		case <-ticker.C:
			CleanupOldChunks(basePath, retentionDays)
		}
	}
}

// CleanupOldChunks removes chunk files older than retention period
func CleanupOldChunks(basePath string, retentionDays int) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	deletedCount := 0
	deletedBytes := int64(0)

	log.Printf("[RetentionWorker] Starting cleanup, cutoff: %s", cutoff.Format(time.RFC3339))

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking on error
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file is older than cutoff
		if info.ModTime().Before(cutoff) {
			size := info.Size()
			if err := os.Remove(path); err != nil {
				log.Printf("[RetentionWorker] Failed to delete %s: %v", path, err)
				return nil
			}
			deletedCount++
			deletedBytes += size
			log.Printf("[RetentionWorker] Deleted old file: %s (age: %v)", 
				filepath.Base(path), time.Since(info.ModTime()).Hours()/24)
		}

		return nil
	})

	if err != nil {
		log.Printf("[RetentionWorker] Cleanup error: %v", err)
	}

	if deletedCount > 0 {
		log.Printf("[RetentionWorker] Cleanup complete: deleted %d files (%.2f MB)", 
			deletedCount, float64(deletedBytes)/1024/1024)
	} else {
		log.Printf("[RetentionWorker] Cleanup complete: no old files to delete")
	}

	// Remove empty directories
	cleanupEmptyDirs(basePath)
}

// cleanupEmptyDirs removes empty directories recursively
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
			if err := os.Remove(path); err == nil {
				log.Printf("[RetentionWorker] Removed empty directory: %s", path)
			}
		}

		return nil
	})
}


