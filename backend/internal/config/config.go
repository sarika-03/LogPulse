package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Storage  StorageConfig  `yaml:"storage"`
	Ingest   IngestConfig   `yaml:"ingest"`
	Auth     AuthConfig     `yaml:"auth"`
	Shutdown ShutdownConfig `yaml:"shutdown"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

type StorageConfig struct {
	Path           string `yaml:"path"`
	ChunkSizeBytes int    `yaml:"chunk_size_bytes"`
	RetentionDays  int    `yaml:"retention_days"`
}

type IngestConfig struct {
	BufferSize    int `yaml:"buffer_size"`
	FlushInterval int `yaml:"flush_interval_ms"`
}

type AuthConfig struct {
	Enabled bool   `yaml:"enabled"`
	APIKey  string `yaml:"api_key"`
}

type ShutdownConfig struct {
	HTTPTimeout     int `yaml:"http_timeout_seconds"`
	IngestorTimeout int `yaml:"ingestor_timeout_seconds"`
	ProgressLog     int `yaml:"progress_log_interval_seconds"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		// Return default config if file not found
		return DefaultConfig(), nil
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Validate and clamp shutdown timeouts to prevent panics
	if cfg.Shutdown.HTTPTimeout <= 0 {
		cfg.Shutdown.HTTPTimeout = 30 // Default to 30 seconds
	}
	if cfg.Shutdown.IngestorTimeout <= 0 {
		cfg.Shutdown.IngestorTimeout = 30 // Default to 30 seconds
	}
	if cfg.Shutdown.ProgressLog <= 0 {
		cfg.Shutdown.ProgressLog = 2 // Default to 2 seconds
	}

	// Override with environment variables
	if port := os.Getenv("LOKILITE_PORT"); port != "" {
		cfg.Server.Port = port
	}
	if apiKey := os.Getenv("LOKILITE_API_KEY"); apiKey != "" {
		cfg.Auth.APIKey = apiKey
		cfg.Auth.Enabled = true
	}
	if storagePath := os.Getenv("LOKILITE_STORAGE_PATH"); storagePath != "" {
		cfg.Storage.Path = storagePath
	}

	return &cfg, nil
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: "8080",
		},
		Storage: StorageConfig{
			Path:           "./data/logs",
			ChunkSizeBytes: 1024 * 1024, // 1MB
			RetentionDays:  7,
		},
		Ingest: IngestConfig{
			BufferSize:    1000,
			FlushInterval: 5000,
		},
		Auth: AuthConfig{
			Enabled: false,
			APIKey:  "",
		},
		Shutdown: ShutdownConfig{
			HTTPTimeout:     30,
			IngestorTimeout: 30,
			ProgressLog:     2,
		},
	}
}
