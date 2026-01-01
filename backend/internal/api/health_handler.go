package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/logpulse/backend/internal/index"
	"github.com/logpulse/backend/internal/ingest"
	"github.com/logpulse/backend/internal/storage"
)

var startTime = time.Now()

// HealthHandler handles health and metrics endpoints
type HealthHandler struct {
	ingestor  *ingest.Ingestor
	reader    *storage.Reader
	index     *index.Index
	writer    *storage.Writer
	streamHub *StreamHub
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(ingestor *ingest.Ingestor, reader *storage.Reader, idx *index.Index) *HealthHandler {
	return &HealthHandler{
		ingestor: ingestor,
		reader:   reader,
		index:    idx,
	}
}

// SetWriter sets the storage writer for metrics
func (h *HealthHandler) SetWriter(w *storage.Writer) {
	h.writer = w
}

// SetStreamHub sets the stream hub for metrics
func (h *HealthHandler) SetStreamHub(hub *StreamHub) {
	h.streamHub = hub
}

// Health handles GET /health
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	lines, _, broadcasts := h.ingestor.GetMetrics()
	chunkCount, _ := h.index.Stats()

	var storageUsed int64
	if h.writer != nil {
		storageUsed = h.writer.GetStorageSize()
	}

	var clientCount int
	var drops int64
	if h.streamHub != nil {
		clientCount = h.streamHub.GetClientCount()
		drops = h.streamHub.GetDroppedMessages()
	}

	uptime := int64(time.Since(startTime).Seconds())

	// Calculate ingestion rate (simple approximation)
	ingestionRate := int(0)
	if uptime > 0 {
		ingestionRate = int(lines / uptime)
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
		"status": "healthy",
		"ingestionRate": %d,
		"storageUsed": %d,
		"chunksCount": %d,
		"uptime": %d,
		"streamClients": %d,
		"broadcastedLines": %d,
		"droppedMessages": %d
	}`, ingestionRate, storageUsed, chunkCount, uptime, clientCount, broadcasts, drops)
}

// Metrics handles GET /metrics (Prometheus format)
func (h *HealthHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	lines, bytes, broadcasts := h.ingestor.GetMetrics()
	chunkCount, _ := h.index.Stats()

	var storageUsed int64
	if h.writer != nil {
		storageUsed = h.writer.GetStorageSize()
	}

	var clientCount int
	var drops int64
	if h.streamHub != nil {
		clientCount = h.streamHub.GetClientCount()
		drops = h.streamHub.GetDroppedMessages()
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	// Expose Kubernetes labels/annotations if present
	k8sLabels, k8sAnnotations := h.ingestor.GetK8sContext()
	for k, v := range k8sLabels {
		fmt.Fprintf(w, "# HELP lokiclone_k8s_label_%s Kubernetes label %s\n", k, k)
		fmt.Fprintf(w, "# TYPE lokiclone_k8s_label_%s gauge\n", k)
		fmt.Fprintf(w, "lokiclone_k8s_label_%s{value=\"%s\"} 1\n\n", k, v)
	}
	for k, v := range k8sAnnotations {
		fmt.Fprintf(w, "# HELP lokiclone_k8s_annotation_%s Kubernetes annotation %s\n", k, k)
		fmt.Fprintf(w, "# TYPE lokiclone_k8s_annotation_%s gauge\n", k)
		fmt.Fprintf(w, "lokiclone_k8s_annotation_%s{value=\"%s\"} 1\n\n", k, v)
	}

	fmt.Fprintf(w, `# HELP lokiclone_ingested_bytes_total Total bytes ingested
# TYPE lokiclone_ingested_bytes_total counter
lokiclone_ingested_bytes_total %d

# HELP lokiclone_ingested_lines_total Total log lines ingested
# TYPE lokiclone_ingested_lines_total counter
lokiclone_ingested_lines_total %d

# HELP lokiclone_broadcasted_lines_total Total log lines broadcasted to stream subscribers
# TYPE lokiclone_broadcasted_lines_total counter
lokiclone_broadcasted_lines_total %d

# HELP lokiclone_dropped_broadcasts_total Total broadcast messages dropped
# TYPE lokiclone_dropped_broadcasts_total counter
lokiclone_dropped_broadcasts_total %d

# HELP lokiclone_stream_clients_connected Current number of connected stream clients
# TYPE lokiclone_stream_clients_connected gauge
lokiclone_stream_clients_connected %d

# HELP lokiclone_chunks_stored_total Total chunks stored on disk
# TYPE lokiclone_chunks_stored_total gauge
lokiclone_chunks_stored_total %d

# HELP lokiclone_storage_bytes Total storage used in bytes
# TYPE lokiclone_storage_bytes gauge
lokiclone_storage_bytes %d

# HELP lokiclone_uptime_seconds Server uptime in seconds
# TYPE lokiclone_uptime_seconds gauge
lokiclone_uptime_seconds %d
`, bytes, lines, broadcasts, drops, clientCount, chunkCount, storageUsed, int64(time.Since(startTime).Seconds()))
}
