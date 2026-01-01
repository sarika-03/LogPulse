package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gorilla/websocket"
	"github.com/logpulse/backend/internal/models"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// StreamHub manages WebSocket connections for live streaming
type StreamHub struct {
	clients      map[*websocket.Conn]StreamFilter
	register     chan *clientRegistration
	unregister   chan *websocket.Conn
	broadcast    chan *models.LogEntry
	mu           sync.RWMutex
	dropCount    int64 // Track dropped messages
	broadcastErr chan error
	doneChan     chan struct{}
}

type clientRegistration struct {
	conn   *websocket.Conn
	filter StreamFilter
}

type StreamFilter struct {
	Labels map[string]string `json:"labels"`
}

// NewStreamHub creates a new streaming hub
func NewStreamHub() *StreamHub {
	return &StreamHub{
		clients:      make(map[*websocket.Conn]StreamFilter),
		register:     make(chan *clientRegistration, 100),
		unregister:   make(chan *websocket.Conn, 100),
		broadcast:    make(chan *models.LogEntry, 5000), // Increased buffer
		dropCount:    0,
		broadcastErr: make(chan error, 100),
		doneChan:     make(chan struct{}),
	}
}

// Run starts the hub's main loop
func (h *StreamHub) Run() {
	log.Println("[StreamHub] Starting hub")
	defer func() {
		close(h.doneChan)
		log.Println("[StreamHub] Hub stopped")
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case reg := <-h.register:
			h.mu.Lock()
			h.clients[reg.conn] = reg.filter
			clientCount := len(h.clients)
			h.mu.Unlock()
			log.Printf("[StreamHub] Client connected with filter %v. Total: %d", reg.filter.Labels, clientCount)

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				clientCount := len(h.clients)
				h.mu.Unlock()
				log.Printf("[StreamHub] Client disconnected. Total: %d", clientCount)
				conn.Close()
			} else {
				h.mu.Unlock()
			}

		case entry := <-h.broadcast:
			// Process broadcast with separate lock for reading clients
			h.mu.RLock()
			clientCount := len(h.clients)
			clientsCopy := make([](*websocket.Conn), 0, clientCount)
			filtersCopy := make([]StreamFilter, 0, clientCount)
			
			for conn, filter := range h.clients {
				clientsCopy = append(clientsCopy, conn)
				filtersCopy = append(filtersCopy, filter)
			}
			h.mu.RUnlock()

			// Send to matching clients
			failedConns := make([]*websocket.Conn, 0)
			for i, conn := range clientsCopy {
				filter := filtersCopy[i]
				
				// Check if log matches client's filter
				if matchesFilter(entry.Labels, filter.Labels) {
					msg, _ := json.Marshal(map[string]interface{}{
						"type": "log",
						"data": map[string]interface{}{
							"id":        entry.ID,
							"timestamp": entry.Timestamp.Format(time.RFC3339Nano),
							"message":   entry.Line,
							"labels":    entry.Labels,
							"level":     entry.Labels["level"],
						},
					})

					// Non-blocking write with timeout
					done := make(chan error, 1)
					go func(c *websocket.Conn, m []byte) {
						c.SetWriteDeadline(time.Now().Add(5 * time.Second))
						done <- c.WriteMessage(websocket.TextMessage, m)
					}(conn, msg)

					select {
					case err := <-done:
						if err != nil {
							log.Printf("[StreamHub] Failed to write to client: %v", err)
							failedConns = append(failedConns, conn)
						}
					case <-time.After(6 * time.Second):
						log.Printf("[StreamHub] Client write timeout")
						failedConns = append(failedConns, conn)
					}
				}
			}

			// Unregister failed connections
			for _, conn := range failedConns {
				h.unregister <- conn
			}

		case <-ticker.C:
			drops := atomic.LoadInt64(&h.dropCount)
			h.mu.RLock()
			clientCount := len(h.clients)
			h.mu.RUnlock()
			if clientCount > 0 || drops > 0 {
				log.Printf("[StreamHub] Status - Clients: %d, Drops: %d, QueueLen: %d/%d",
					clientCount, drops, len(h.broadcast), cap(h.broadcast))
			}
		}
	}
}

// Broadcast sends a log entry to all matching clients (non-blocking)
func (h *StreamHub) Broadcast(entry *models.LogEntry) {
	select {
	case h.broadcast <- entry:
		// Successfully queued
	default:
		// Channel full, drop message and track
		drops := atomic.AddInt64(&h.dropCount, 1)
		if drops%100 == 0 { // Log every 100 drops to avoid spam
			log.Printf("[StreamHub] WARN: Broadcast channel full, dropping message. Total drops: %d", drops)
		}
	}
}

// matchesFilter checks if log labels match the filter
func matchesFilter(logLabels, filterLabels map[string]string) bool {
	if len(filterLabels) == 0 {
		return true // No filter means match all
	}
	for k, v := range filterLabels {
		if logLabels[k] != v {
			return false
		}
	}
	return true
}

// StreamHandler handles WebSocket connections for live log streaming
type StreamHandler struct {
	hub *StreamHub
}

// ServeMetricsSSE handles /metrics/stream SSE endpoint for real-time Prometheus metrics
func ServeMetricsSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			// Capture Prometheus metrics as text
			w.Write([]byte("event: metrics\n"))
			w.Write([]byte("data: "))
			promhttp.Handler().ServeHTTP(&sseWriter{w}, r)
			w.Write([]byte("\n\n"))
			flusher.Flush()
		}
	}
}

// sseWriter wraps http.ResponseWriter to capture promhttp output as SSE data
type sseWriter struct {
	http.ResponseWriter
}

func (w *sseWriter) Write(p []byte) (int, error) {
	// Replace newlines with \ndata:  for SSE compliance
	s := string(p)
	s = s[:len(s)-1] // Remove last newline
	lines := []byte("")
	for _, line := range splitLines(s) {
		lines = append(lines, []byte("\ndata: "+line)...)
	}
	return w.ResponseWriter.Write(lines)
}

func splitLines(s string) []string {
       var lines []string
       start := 0
       for i := 0; i < len(s); i++ {
	       if s[i] == '\n' {
		       lines = append(lines, s[start:i])
		       start = i + 1
	       }
       }
       if start < len(s) {
	       lines = append(lines, s[start:])
       }
       return lines
}

// NewStreamHandler creates a new stream handler
func NewStreamHandler(hub *StreamHub) *StreamHandler {
	return &StreamHandler{hub: hub}
}

// HandleStream handles GET /stream WebSocket endpoint
func (h *StreamHandler) HandleStream(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[StreamHandler] WebSocket upgrade error: %v", err)
		return
	}

	// Parse filter from query params
	filter := StreamFilter{
		Labels: make(map[string]string),
	}

	// Get labels from query string
	for key, values := range r.URL.Query() {
		if key != "query" && len(values) > 0 {
			filter.Labels[key] = values[0]
		}
	}

	// Register client
	h.hub.register <- &clientRegistration{
		conn:   conn,
		filter: filter,
	}

	// Send welcome message
	welcome, _ := json.Marshal(map[string]interface{}{
		"type":    "connected",
		"message": "Connected to log stream",
		"filter":  filter.Labels,
	})
	conn.WriteMessage(websocket.TextMessage, welcome)

	// Context for goroutine
	done := make(chan struct{})

	// Handle incoming messages (for filter updates)
	go func() {
		defer close(done)
		defer func() {
			h.hub.unregister <- conn
			log.Printf("[StreamHandler] Reader goroutine exited for client with filter %v", filter.Labels)
		}()

		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("[StreamHandler] WebSocket error: %v", err)
				}
				return
			}

			// Handle filter update messages
			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("[StreamHandler] Failed to unmarshal message: %v", err)
				continue
			}

			if msg["type"] == "filter" {
				if labels, ok := msg["labels"].(map[string]interface{}); ok {
					newFilter := StreamFilter{Labels: make(map[string]string)}
					for k, v := range labels {
						if str, ok := v.(string); ok {
							newFilter.Labels[k] = str
						}
					}
					h.hub.mu.Lock()
					h.hub.clients[conn] = newFilter
					h.hub.mu.Unlock()

					// Confirm filter update
					confirm, _ := json.Marshal(map[string]interface{}{
						"type":   "filter_updated",
						"filter": newFilter.Labels,
					})
					conn.WriteMessage(websocket.TextMessage, confirm)
					log.Printf("[StreamHandler] Filter updated: %v", newFilter.Labels)
				}
			}

			// Reset read deadline on successful message
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		}
	}()

	// Keep the connection alive with periodic pings
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.Printf("[StreamHandler] Ping error: %v", err)
				h.hub.unregister <- conn
				return
			}
		}
	}
}

// GetClientCount returns the number of connected clients
func (h *StreamHub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetDroppedMessages returns the count of dropped broadcast messages
func (h *StreamHub) GetDroppedMessages() int64 {
	return atomic.LoadInt64(&h.dropCount)
}

// ResetDropCounter resets the dropped message counter
func (h *StreamHub) ResetDropCounter() {
	atomic.StoreInt64(&h.dropCount, 0)
}
