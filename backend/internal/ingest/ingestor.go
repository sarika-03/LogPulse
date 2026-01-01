package ingest

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/logpulse/backend/internal/index"
	"github.com/logpulse/backend/internal/models"
	"github.com/logpulse/backend/internal/storage"
)

// StreamBroadcaster interface for live log streaming
type StreamBroadcaster interface {
	Broadcast(entry *models.LogEntry)
}

// Ingestor handles incoming logs and buffers them before writing
type Ingestor struct {
	index       *index.Index
	writer      *storage.Writer
	broadcaster StreamBroadcaster
	bufSize     int

	// Buffer per label set
	buffers  map[string]*logBuffer
	bufferMu sync.Mutex

	// Metrics
	ingestedLines int64
	ingestedBytes int64
	broadcastedLines int64
	metricsMu     sync.RWMutex

	// Kubernetes context
	k8sLabels      map[string]string
	k8sAnnotations map[string]string

	stopChan chan struct{}
	wg       sync.WaitGroup
}

	// GetK8sContext returns Kubernetes labels and annotations
	func (ing *Ingestor) GetK8sContext() (map[string]string, map[string]string) {
		ing.metricsMu.RLock()
		defer ing.metricsMu.RUnlock()
		return ing.k8sLabels, ing.k8sAnnotations
	}

type logBuffer struct {
	labels  map[string]string
	entries []models.LogEntry
	size    int
}

// NewIngestor creates a new log ingestor
func NewIngestor(idx *index.Index, writer *storage.Writer, bufferSize int, broadcaster StreamBroadcaster) *Ingestor {
	return &Ingestor{
		index:       idx,
		writer:      writer,
		broadcaster: broadcaster,
		bufSize:     bufferSize,
		buffers:     make(map[string]*logBuffer),
		k8sLabels:      make(map[string]string),
		k8sAnnotations: make(map[string]string),
		stopChan:    make(chan struct{}),
	}
}

// Start begins the background flush worker
func (ing *Ingestor) Start() {
	ing.wg.Add(1)
	go ing.flushWorker()
}

// Stop gracefully shuts down the ingestor
func (ing *Ingestor) Stop() {
	close(ing.stopChan)
	ing.wg.Wait()
	ing.flushAll()
}

// Ingest processes incoming log streams
// ExtractK8sContext extracts Kubernetes labels/annotations from log labels
func ExtractK8sContext(labels map[string]string) (map[string]string, map[string]string) {
	k8sLabels := make(map[string]string)
	k8sAnnotations := make(map[string]string)
	for k, v := range labels {
		if len(k) > 4 && k[:4] == "k8s_" {
			k8sLabels[k[4:]] = v
		}
		if len(k) > 10 && k[:10] == "k8s_annot_" {
			k8sAnnotations[k[10:]] = v
		}
	}
	return k8sLabels, k8sAnnotations
}

func (ing *Ingestor) Ingest(req *models.IngestRequest) (int, error) {
	accepted := 0

	for _, stream := range req.Streams {
		// Extract and store Kubernetes context if present
		k8sLabels, k8sAnnotations := ExtractK8sContext(stream.Labels)
		if len(k8sLabels) > 0 {
			ing.k8sLabels = k8sLabels
		}
		if len(k8sAnnotations) > 0 {
			ing.k8sAnnotations = k8sAnnotations
		}
		if err := ValidateStream(&stream); err != nil {
			log.Printf("[Ingestor] Invalid stream: %v", err)
			continue
		}

		labelHash := models.Labels(stream.Labels).Hash()

		ing.bufferMu.Lock()
		buf, exists := ing.buffers[labelHash]
		if !exists {
			buf = &logBuffer{
				labels:  stream.Labels,
				entries: make([]models.LogEntry, 0, ing.bufSize),
			}
			ing.buffers[labelHash] = buf
		}

		for _, entry := range stream.Entries {
			ts, err := time.Parse(time.RFC3339, entry.Ts)
			if err != nil {
				ts = time.Now()
			}

			logEntry := models.LogEntry{
				ID:        generateLogID(),
				Timestamp: ts,
				Line:      entry.Line,
				Labels:    stream.Labels,
			}

			buf.entries = append(buf.entries, logEntry)
			buf.size += len(entry.Line)
			accepted++

			// Broadcast to live stream subscribers (non-blocking)
			if ing.broadcaster != nil {
				// Use non-blocking broadcast to avoid deadlocking ingest pipeline
				go func(entry *models.LogEntry) {
					ing.broadcaster.Broadcast(entry)
					atomic.AddInt64(&ing.broadcastedLines, 1)
				}(&logEntry)
				
				log.Printf("[Ingestor] Broadcast log entry: ID=%s, Labels=%v, Message=%.50s", 
					logEntry.ID, stream.Labels, logEntry.Line)
			}

			// Update metrics
			ing.metricsMu.Lock()
			ing.ingestedLines++
			ing.ingestedBytes += int64(len(entry.Line))
			ing.metricsMu.Unlock()
		}

		// Flush if buffer is full
		if len(buf.entries) >= ing.bufSize {
			ing.flushBuffer(labelHash, buf)
			ing.buffers[labelHash] = &logBuffer{
				labels:  stream.Labels,
				entries: make([]models.LogEntry, 0, ing.bufSize),
			}
		}
		ing.bufferMu.Unlock()
	}

	log.Printf("[Ingestor] Ingest request processed: accepted=%d, totalLines=%d", 
		accepted, atomic.LoadInt64(&ing.ingestedLines))

	return accepted, nil
}

// flushWorker periodically flushes buffers
func (ing *Ingestor) flushWorker() {
	defer ing.wg.Done()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ing.flushAll()
		case <-ing.stopChan:
			return
		}
	}
}

// flushAll flushes all buffers
func (ing *Ingestor) flushAll() {
	ing.bufferMu.Lock()
	defer ing.bufferMu.Unlock()

	for hash, buf := range ing.buffers {
		if len(buf.entries) > 0 {
			ing.flushBuffer(hash, buf)
			buf.entries = buf.entries[:0]
			buf.size = 0
		}
	}
}

// flushBuffer writes a buffer to disk
func (ing *Ingestor) flushBuffer(hash string, buf *logBuffer) {
	if len(buf.entries) == 0 {
		return
	}

	startTime := time.Now()
	chunkID, startTs, endTs, err := ing.writer.WriteChunk(buf.labels, buf.entries)
	if err != nil {
		log.Printf("[Ingestor] ERROR failed to write chunk: %v", err)
		return
	}

	ing.index.AddChunk(chunkID, buf.labels, startTs, endTs, len(buf.entries))
	elapsed := time.Since(startTime)
	log.Printf("[Ingestor] Flushed chunk: ID=%s, entries=%d, labels=%v, duration=%v",
		chunkID, len(buf.entries), buf.labels, elapsed)
}

// GetMetrics returns ingestion metrics
func (ing *Ingestor) GetMetrics() (lines int64, bytes int64, broadcasts int64) {
	ing.metricsMu.RLock()
	defer ing.metricsMu.RUnlock()
	return ing.ingestedLines, ing.ingestedBytes, atomic.LoadInt64(&ing.broadcastedLines)
}

// generateLogID creates a unique log ID
func generateLogID() string {
	return time.Now().Format("20060102150405.000000000")
}
