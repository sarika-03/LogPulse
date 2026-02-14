package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/logpulse/backend/internal/api"
	"github.com/logpulse/backend/internal/config"
	"github.com/logpulse/backend/internal/index"
	"github.com/logpulse/backend/internal/ingest"
	"github.com/logpulse/backend/internal/plugin"
	"github.com/logpulse/backend/internal/query"
	"github.com/logpulse/backend/internal/storage"
	gootel "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func main() {
	// Create root context for graceful shutdown
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	// Load alert rules
	var webhookNotifier *plugin.WebhookNotifier
	webhookCfgs, err := config.LoadWebhooks("configs/webhooks.yaml")
	if err == nil && len(webhookCfgs) > 0 {
		pluginCfgs := make([]plugin.WebhookConfig, len(webhookCfgs))
		for i, w := range webhookCfgs {
			pluginCfgs[i] = plugin.WebhookConfig{URL: w.URL, Events: w.Events}
		}
		webhookNotifier = plugin.NewWebhookNotifier(pluginCfgs)
		log.Printf("Loaded %d webhook(s)", len(pluginCfgs))
	}

	alertRules, _ := config.LoadAlerts("configs/alerts.yaml")
	alertManager := plugin.NewAlertManager(webhookNotifier)
	for _, rule := range alertRules {
		alertManager.AddRule(plugin.AlertRule{
			Name:      rule.Name,
			Expr:      rule.Expr,
			Threshold: rule.Threshold,
			Window:    5 * time.Minute,
			Channels:  rule.Channels,
			Labels:    rule.Labels,
		})
	}

	// Proper query function for alert evaluation
	var executor *query.Executor
	queryFunc := func(expr string) (float64, error) {
		if executor == nil {
			return 0, nil
		}
		endTime := time.Now()
		startTime := endTime.Add(-5 * time.Minute)
		result, err := executor.Execute(expr, startTime, endTime, 0)
		if err != nil {
			return 0, err
		}
		if result.Aggregation != nil {
			return result.Aggregation.Value, nil
		}
		return float64(result.Stats.MatchedLines), nil
	}

	// Alert evaluation with context cancellation
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-rootCtx.Done():
				log.Println("[AlertEvaluator] Shutting down")
				return
			case <-ticker.C:
				alertManager.EvaluateRules(queryFunc)
			}
		}
	}()

	// --- OpenTelemetry Tracing Setup ---
	exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatalf("Failed to create OTel exporter: %v", err)
	}
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("insight-stream-backend"),
		)),
	)
	gootel.SetTracerProvider(tp)
	defer func() { _ = tp.Shutdown(context.Background()) }()

	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting LokiLite server on port %s", cfg.Server.Port)

	// Initialize components
	labelIndex := index.NewIndex()
	storageWriter := storage.NewWriter(cfg.Storage.Path, cfg.Storage.ChunkSizeBytes)
	storageReader := storage.NewReader(cfg.Storage.Path)

	// Initialize executor for alerts
	executor = query.NewExecutor(labelIndex, storageReader)

	// Initialize streaming hub with context
	streamHub := api.NewStreamHub()
	go streamHub.Run(rootCtx)

	// Initialize ingestor with stream hub for live broadcasting
	ingestor := ingest.NewIngestor(labelIndex, storageWriter, cfg.Ingest.BufferSize, streamHub)

	// Start background workers with context
	go ingestor.Start()
	go storage.StartRetentionWorker(rootCtx, cfg.Storage.Path, cfg.Storage.RetentionDays)

	// Setup HTTP server
	router := api.NewRouterWithWebhooks(ingestor, storageReader, labelIndex, cfg, streamHub, webhookNotifier)

	// Create health handler and set up streaming metrics
	healthHandler := api.NewHealthHandler(ingestor, storageReader, labelIndex)
	healthHandler.SetWriter(storageWriter)
	healthHandler.SetStreamHub(streamHub)

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Proper graceful shutdown with context and synchronization
	shutdownComplete := make(chan struct{})

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Graceful shutdown initiated...")

		// Step 1: Shutdown HTTP server first to drain in-flight requests
		// This allows existing requests to complete before we stop accepting new ones
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		log.Println("Draining in-flight HTTP requests...")
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		} else {
			log.Println("HTTP server shutdown complete - all requests drained")
		}

		// Step 2: Flush ingestor buffers to ensure all ingested logs are written
		flushDone := make(chan struct{})
		go func() {
			log.Println("Flushing ingestor buffers...")
			ingestor.Stop()
			close(flushDone)
		}()

		select {
		case <-flushDone:
			log.Println("Ingestor flushed successfully")
		case <-time.After(10 * time.Second):
			log.Println("WARNING: Ingestor flush timeout")
		}

		// Step 3: Cancel context to stop background workers (alerts, retention, etc.)
		log.Println("Stopping background workers...")
		rootCancel()

		close(shutdownComplete)
	}()

	// Start server
	log.Printf("LokiLite is ready at http://localhost:%s", cfg.Server.Port)
	log.Printf("WebSocket streaming available at ws://localhost:%s/stream", cfg.Server.Port)

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	// Wait for graceful shutdown to complete
	<-shutdownComplete
	log.Println("Server stopped cleanly")
}
