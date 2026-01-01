package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/logpulse/backend/internal/index"
	"github.com/logpulse/backend/internal/query"
	"github.com/logpulse/backend/internal/storage"
)

// QueryHandler handles log queries
type QueryHandler struct {
	index    *index.Index
	reader   *storage.Reader
	executor *query.Executor
}

// NewQueryHandler creates a new query handler
func NewQueryHandler(idx *index.Index, reader *storage.Reader) *QueryHandler {
	return &QueryHandler{
		index:    idx,
		reader:   reader,
		executor: query.NewExecutor(idx, reader),
	}
}

// Query handles GET /query
func (h *QueryHandler) Query(w http.ResponseWriter, r *http.Request) {
	queryStr := r.URL.Query().Get("query")
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	limitStr := r.URL.Query().Get("limit")

	// Parse time range
	var startTime, endTime time.Time
	var err error

	if startStr != "" {
		startTime, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			http.Error(w, "Invalid start time format", http.StatusBadRequest)
			return
		}
	} else {
		startTime = time.Now().Add(-1 * time.Hour)
	}

	if endStr != "" {
		endTime, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			http.Error(w, "Invalid end time format", http.StatusBadRequest)
			return
		}
	} else {
		endTime = time.Now()
	}

	// Parse limit
	limit := 100
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			limit = 100
		}
	}

	// Execute query
	result, err := h.executor.Execute(queryStr, startTime, endTime, limit)
	if err != nil {
		http.Error(w, "Query error: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Labels handles GET /labels
func (h *QueryHandler) Labels(w http.ResponseWriter, r *http.Request) {
	labels := h.index.GetAllLabels()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(labels)
}

// LabelValues handles GET /labels/{name}/values
func (h *QueryHandler) LabelValues(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	labelName := vars["name"]

	values := h.index.GetLabelValues(labelName)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(values)
}
