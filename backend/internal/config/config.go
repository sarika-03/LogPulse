package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Storage StorageConfig `yaml:"storage"`
	Ingest  IngestConfig  `yaml:"ingest"`
	Auth    AuthConfig    `yaml:"auth"`
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
	}
}
