package api

import (
	"net/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gorilla/mux"
	"github.com/logpulse/backend/internal/config"
	"github.com/logpulse/backend/internal/index"
	"github.com/logpulse/backend/internal/ingest"
	"github.com/logpulse/backend/internal/plugin"
	"github.com/logpulse/backend/internal/storage"
)

// NewRouterWithWebhooks configures the main HTTP router.
func NewRouterWithWebhooks(
	ingestor *ingest.Ingestor,
	reader *storage.Reader,
	labelIndex *index.Index,
	cfg *config.Config,
	streamHub *StreamHub,
	webhookNotifier interface{},
) *mux.Router {
	router := mux.NewRouter()

	healthHandler := NewHealthHandler(ingestor, reader, labelIndex)
	var ingestHandler *IngestHandler
	if webhookNotifier != nil {
		ingestHandler = NewIngestHandler(ingestor, webhookNotifier.(*plugin.WebhookNotifier))
	} else {
		       ingestHandler = NewIngestHandler(ingestor, nil)
	       }
	queryHandler := NewQueryHandler(labelIndex, reader)
	streamHandler := NewStreamHandler(streamHub)
	lokiHandler := NewLokiHandler(labelIndex, reader)
	alertHandler := NewAlertHandler()

	router.Use(corsMiddleware)
	router.Use(loggingMiddleware)

	if cfg.Auth.Enabled {
		 router.Use(authMiddleware(cfg.Auth.APIKey))
	}

	router.HandleFunc("/health", healthHandler.Health).Methods("GET", "OPTIONS")
	router.HandleFunc("/metrics", healthHandler.Metrics).Methods("GET", "OPTIONS")
	router.Handle("/prometheus-metrics", promhttp.Handler()).Methods("GET")
	router.HandleFunc("/metrics/stream", ServeMetricsSSE).Methods("GET")

	router.HandleFunc("/ingest", ingestHandler.Ingest).Methods("POST", "OPTIONS")

	router.HandleFunc("/query", queryHandler.Query).Methods("GET", "OPTIONS")
	router.HandleFunc("/labels", queryHandler.Labels).Methods("GET", "OPTIONS")
	router.HandleFunc("/labels/{name}/values", queryHandler.LabelValues).Methods("GET", "OPTIONS")

	// WebSocket for live tailing
	router.HandleFunc("/stream", streamHandler.HandleStream).Methods("GET")


	router.HandleFunc("/alerts", alertHandler.GetAlerts).Methods("GET", "OPTIONS")
	router.HandleFunc("/alerts", alertHandler.CreateAlert).Methods("POST", "OPTIONS")
	router.HandleFunc("/alerts/{id}", alertHandler.GetAlert).Methods("GET", "OPTIONS")
	router.HandleFunc("/alerts/{id}", alertHandler.UpdateAlert).Methods("PUT", "OPTIONS")
	router.HandleFunc("/alerts/{id}", alertHandler.DeleteAlert).Methods("DELETE", "OPTIONS")
	router.HandleFunc("/alerts/{id}/status", alertHandler.UpdateAlertStatus).Methods("PATCH", "OPTIONS")

	// Loki-compatible API for Grafana
	router.HandleFunc("/ready", lokiHandler.Ready).Methods("GET", "OPTIONS")
	router.HandleFunc("/loki/api/v1/query_range", lokiHandler.QueryRange).Methods("GET", "OPTIONS")
	router.HandleFunc("/loki/api/v1/query", lokiHandler.Query).Methods("GET", "OPTIONS")
	router.HandleFunc("/loki/api/v1/labels", lokiHandler.Labels).Methods("GET", "OPTIONS")
	router.HandleFunc("/loki/api/v1/label/{name}/values", lokiHandler.LabelValues).Methods("GET", "OPTIONS")

	return router
}

// For backward compatibility
func NewRouter(
	ingestor *ingest.Ingestor,
	reader *storage.Reader,
	labelIndex *index.Index,
	cfg *config.Config,
	streamHub *StreamHub,
) *mux.Router {
	return NewRouterWithWebhooks(ingestor, reader, labelIndex, cfg, streamHub, nil)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func authMiddleware(apiKey string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}

			// Skip auth for WebSocket upgrade
			if r.Header.Get("Upgrade") == "websocket" {
				next.ServeHTTP(w, r)
				return
			}

			key := r.Header.Get("X-API-Key")
			if key == "" {
				key = r.Header.Get("Authorization")
			}

			if key != apiKey {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

