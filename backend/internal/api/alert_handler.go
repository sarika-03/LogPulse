package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// AlertRule represents an alert configuration
type AlertRule struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Query     string    `json:"query"`
	Condition string    `json:"condition"` // gt, lt, eq, gte, lte
	Threshold int       `json:"threshold"`
	Duration  string    `json:"duration"` // 5m, 1h, etc
	Severity  string    `json:"severity"` // critical, warning, info
	Enabled   bool      `json:"enabled"`
	Webhook   string    `json:"webhook,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AlertHandler handles alert endpoints
type AlertHandler struct {
	mu     sync.RWMutex
	alerts map[string]*AlertRule
}

// NewAlertHandler creates a new alert handler
func NewAlertHandler() *AlertHandler {
	return &AlertHandler{
		alerts: make(map[string]*AlertRule),
	}
}

// GetAlerts returns all alerts
func (h *AlertHandler) GetAlerts(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	alerts := make([]*AlertRule, 0, len(h.alerts))
	for _, alert := range h.alerts {
		alerts = append(alerts, alert)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(alerts)
}

// CreateAlert creates a new alert
func (h *AlertHandler) CreateAlert(w http.ResponseWriter, r *http.Request) {
	var req AlertRule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Validation
	if req.Name == "" {
		http.Error(w, "Alert name is required", http.StatusBadRequest)
		return
	}
	if req.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}
	if req.Condition == "" {
		http.Error(w, "Condition is required", http.StatusBadRequest)
		return
	}
	if req.Threshold < 0 {
		http.Error(w, "Threshold must be >= 0", http.StatusBadRequest)
		return
	}
	if req.Duration == "" {
		http.Error(w, "Duration is required", http.StatusBadRequest)
		return
	}
	if req.Severity == "" {
		req.Severity = "warning"
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Generate UUID
	b := make([]byte, 16)
	rand.Read(b)
	id := hex.EncodeToString(b)

	alert := &AlertRule{
		ID:        id,
		Name:      req.Name,
		Query:     req.Query,
		Condition: req.Condition,
		Threshold: req.Threshold,
		Duration:  req.Duration,
		Severity:  req.Severity,
		Enabled:   true,
		Webhook:   req.Webhook,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	h.alerts[alert.ID] = alert

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(alert)
}

// GetAlert returns a specific alert
func (h *AlertHandler) GetAlert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	h.mu.RLock()
	alert, exists := h.alerts[id]
	h.mu.RUnlock()

	if !exists {
		http.Error(w, "Alert not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(alert)
}

// UpdateAlert updates an alert
func (h *AlertHandler) UpdateAlert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req AlertRule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	alert, exists := h.alerts[id]
	if !exists {
		http.Error(w, "Alert not found", http.StatusNotFound)
		return
	}

	// Update fields
	if req.Name != "" {
		alert.Name = req.Name
	}
	if req.Query != "" {
		alert.Query = req.Query
	}
	if req.Condition != "" {
		alert.Condition = req.Condition
	}
	if req.Threshold > 0 {
		alert.Threshold = req.Threshold
	}
	if req.Duration != "" {
		alert.Duration = req.Duration
	}
	if req.Severity != "" {
		alert.Severity = req.Severity
	}
	if req.Webhook != "" {
		alert.Webhook = req.Webhook
	}

	alert.UpdatedAt = time.Now()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(alert)
}

// UpdateAlertStatus updates only the enabled status
func (h *AlertHandler) UpdateAlertStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	alert, exists := h.alerts[id]
	if !exists {
		http.Error(w, "Alert not found", http.StatusNotFound)
		return
	}

	alert.Enabled = req.Enabled
	alert.UpdatedAt = time.Now()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(alert)
}

// DeleteAlert deletes an alert
func (h *AlertHandler) DeleteAlert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.alerts[id]; !exists {
		http.Error(w, "Alert not found", http.StatusNotFound)
		return
	}

	delete(h.alerts, id)
	w.WriteHeader(http.StatusNoContent)
}
