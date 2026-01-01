package api

import (
	"encoding/json"
	"net/http"

	"github.com/logpulse/backend/internal/plugin"

	"github.com/logpulse/backend/internal/ingest"
	"github.com/logpulse/backend/internal/models"
)


type IngestHandler struct {
	ingestor *ingest.Ingestor
	notifier *plugin.WebhookNotifier
}


func NewIngestHandler(ingestor *ingest.Ingestor, notifier *plugin.WebhookNotifier) *IngestHandler {
	return &IngestHandler{ingestor: ingestor, notifier: notifier}
}


func (h *IngestHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	var req models.IngestRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := ingest.ValidateIngestRequest(&req); err != nil {
		http.Error(w, "Validation error: "+err.Error(), http.StatusBadRequest)
		return
	}


	       accepted, err := h.ingestor.Ingest(&req)
	       if err != nil {
		       http.Error(w, "Ingestion error: "+err.Error(), http.StatusInternalServerError)
		       return
	       }

	       // Trigger webhooks
	       if h.notifier != nil {
		       for _, stream := range req.Streams {
			       for _, entry := range stream.Entries {
				       h.notifier.Notify("log", map[string]interface{}{
					       "labels": stream.Labels,
					       "message": entry.Line,
					       "timestamp": entry.Ts,
				       })
			       }
		       }
	       }

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.IngestResponse{
		Accepted: accepted,
	})
}
