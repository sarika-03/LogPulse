package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// AgentConfig holds the agent configuration
type AgentConfig struct {
	Server   ServerConfig   `yaml:"server"`
	Targets  []TargetConfig `yaml:"targets"`
	Positions string        `yaml:"positions_file"`
}

type ServerConfig struct {
	URL    string `yaml:"url"`
	APIKey string `yaml:"api_key"`
}

type TargetConfig struct {
	Path   string            `yaml:"path"`
	Labels map[string]string `yaml:"labels"`
}

// Position tracks file read positions
type Position struct {
	Path   string `json:"path"`
	Offset int64  `json:"offset"`
}

// Agent is the log collection agent
type Agent struct {
	config    *AgentConfig
	client    *http.Client
	positions map[string]int64
	posMu     sync.RWMutex
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

// IngestRequest matches the server's expected format
type IngestRequest struct {
	Streams []Stream `json:"streams"`
}

type Stream struct {
	Labels  map[string]string `json:"labels"`
	Entries []Entry           `json:"entries"`
}

type Entry struct {
	Ts   string `json:"ts"`
	Line string `json:"line"`
}

func main() {
	configPath := flag.String("config", "agent-config.yaml", "Path to agent config file")
	flag.Parse()

	// Load config
	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	agent := NewAgent(config)

	log.Printf("LokiLite Agent starting...")
	log.Printf("Server: %s", config.Server.URL)
	log.Printf("Watching %d target(s)", len(config.Targets))

	// Handle shutdown
	go func() {
		// Wait for interrupt signal
		sigChan := make(chan os.Signal, 1)
		<-sigChan
		log.Println("Shutting down agent...")
		agent.Stop()
	}()

	agent.Start()
}

func loadConfig(path string) (*AgentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		// Return default config if file not found
		return &AgentConfig{
			Server: ServerConfig{
				URL: "http://localhost:8080",
			},
			Targets: []TargetConfig{
				{
					Path: "/var/log/*.log",
					Labels: map[string]string{
						"job": "varlogs",
					},
				},
			},
			Positions: "./positions.json",
		}, nil
	}

	var config AgentConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Override with environment variables
	if url := os.Getenv("LOKILITE_SERVER_URL"); url != "" {
		config.Server.URL = url
	}
	if apiKey := os.Getenv("LOKILITE_API_KEY"); apiKey != "" {
		config.Server.APIKey = apiKey
	}

	return &config, nil
}

// NewAgent creates a new log agent
func NewAgent(config *AgentConfig) *Agent {
	return &Agent{
		config: config,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		positions: make(map[string]int64),
		stopChan:  make(chan struct{}),
	}
}

// Start begins watching all configured targets
func (a *Agent) Start() {
	// Load saved positions
	a.loadPositions()

	// Start a goroutine for each target
	for _, target := range a.config.Targets {
		a.wg.Add(1)
		go a.watchTarget(target)
	}

	// Periodically save positions
	go a.positionSaver()

	a.wg.Wait()
}

// Stop gracefully stops the agent
func (a *Agent) Stop() {
	close(a.stopChan)
	a.savePositions()
}

// watchTarget watches a single target path (supports glob patterns)
func (a *Agent) watchTarget(target TargetConfig) {
	defer a.wg.Done()

	log.Printf("Watching: %s with labels %v", target.Path, target.Labels)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopChan:
			return
		case <-ticker.C:
			// Expand glob pattern
			files, err := filepath.Glob(target.Path)
			if err != nil {
				log.Printf("Glob error for %s: %v", target.Path, err)
				continue
			}

			for _, file := range files {
				a.tailFile(file, target.Labels)
			}
		}
	}
}

// tailFile reads new lines from a file
func (a *Agent) tailFile(path string, baseLabels map[string]string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}

	// Skip directories
	if info.IsDir() {
		return
	}

	// Get current position
	a.posMu.RLock()
	pos := a.positions[path]
	a.posMu.RUnlock()

	// Handle file truncation (log rotation)
	if info.Size() < pos {
		log.Printf("File %s was truncated, resetting position", path)
		pos = 0
	}

	// Skip if no new data
	if info.Size() == pos {
		return
	}

	// Open file and seek to position
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	if pos > 0 {
		_, err = file.Seek(pos, io.SeekStart)
		if err != nil {
			return
		}
	}

	// Read new lines
	reader := bufio.NewReader(file)
	var entries []Entry
	var bytesRead int64

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("Read error: %v", err)
			break
		}

		bytesRead += int64(len(line))
		lineStr := strings.TrimSpace(string(line))

		if lineStr != "" {
			entries = append(entries, Entry{
				Ts:   time.Now().UTC().Format(time.RFC3339Nano),
				Line: lineStr,
			})
		}
	}

	// Send entries if we have any
	if len(entries) > 0 {
		// Build labels with filename
		labels := make(map[string]string)
		for k, v := range baseLabels {
			labels[k] = v
		}
		labels["filename"] = filepath.Base(path)

		// Auto-detect log level from content
		for i := range entries {
			_ = detectLogLevel(entries[i].Line)
			// Store level in a temporary way - we'll add it to labels per entry
			entries[i].Line = entries[i].Line // Keep original
		}

		// For simplicity, add level to labels based on most common level in batch
		// In production, you'd want per-entry labels
		labels["level"] = detectLogLevel(entries[0].Line)

		a.sendEntries(labels, entries)
	}

	// Update position
	a.posMu.Lock()
	a.positions[path] = pos + bytesRead
	a.posMu.Unlock()
}

// detectLogLevel tries to detect log level from content
func detectLogLevel(line string) string {
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "error") || strings.Contains(lower, "err"):
		return "error"
	case strings.Contains(lower, "warn"):
		return "warn"
	case strings.Contains(lower, "debug"):
		return "debug"
	default:
		return "info"
	}
}

// sendEntries sends log entries to the server
func (a *Agent) sendEntries(labels map[string]string, entries []Entry) {
	req := IngestRequest{
		Streams: []Stream{
			{
				Labels:  labels,
				Entries: entries,
			},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		log.Printf("Marshal error: %v", err)
		return
	}

	httpReq, err := http.NewRequest("POST", a.config.Server.URL+"/ingest", bytes.NewReader(body))
	if err != nil {
		log.Printf("Request creation error: %v", err)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if a.config.Server.APIKey != "" {
		httpReq.Header.Set("X-API-Key", a.config.Server.APIKey)
	}

	resp, err := a.client.Do(httpReq)
	if err != nil {
		log.Printf("Send error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Server returned %d: %s", resp.StatusCode, string(body))
		return
	}

	log.Printf("Sent %d entries from labels %v", len(entries), labels)
}

// loadPositions loads saved file positions
func (a *Agent) loadPositions() {
	data, err := os.ReadFile(a.config.Positions)
	if err != nil {
		return
	}

	var positions []Position
	if err := json.Unmarshal(data, &positions); err != nil {
		return
	}

	a.posMu.Lock()
	for _, p := range positions {
		a.positions[p.Path] = p.Offset
	}
	a.posMu.Unlock()

	log.Printf("Loaded %d saved positions", len(positions))
}

// savePositions saves current file positions
func (a *Agent) savePositions() {
	a.posMu.RLock()
	positions := make([]Position, 0, len(a.positions))
	for path, offset := range a.positions {
		positions = append(positions, Position{Path: path, Offset: offset})
	}
	a.posMu.RUnlock()

	data, err := json.MarshalIndent(positions, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal positions: %v", err)
		return
	}

	if err := os.WriteFile(a.config.Positions, data, 0644); err != nil {
		log.Printf("Failed to save positions: %v", err)
	}
}

// positionSaver periodically saves positions
func (a *Agent) positionSaver() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopChan:
			return
		case <-ticker.C:
			a.savePositions()
		}
	}
}
