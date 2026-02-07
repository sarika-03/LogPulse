package config

import (
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Storage   StorageConfig   `yaml:"storage"`
	Ingest    IngestConfig    `yaml:"ingest"`
	Auth      AuthConfig      `yaml:"auth"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

type ServerConfig struct {
	Port         string `yaml:"port"`
	ReadTimeout  string `yaml:"read_timeout"`
	WriteTimeout string `yaml:"write_timeout"`
	IdleTimeout  string `yaml:"idle_timeout"`
}

type StorageConfig struct {
	Path               string `yaml:"path"`
	ChunkSizeBytes     int    `yaml:"chunk_size_bytes"`
	RetentionDays      int    `yaml:"retention_days"`
	CompressionEnabled bool   `yaml:"compression_enabled"`
}

type IngestConfig struct {
	BufferSize    int `yaml:"buffer_size"`
	FlushInterval int `yaml:"flush_interval_ms"`
	MaxBatchSize  int `yaml:"max_batch_size"`
	Workers       int `yaml:"workers"`
}

type AuthConfig struct {
	Enabled bool   `yaml:"enabled"`
	APIKey  string `yaml:"api_key"`
}

type RateLimitConfig struct {
	Enabled           bool     `yaml:"enabled"`
	RequestsPerMinute int      `yaml:"requests_per_minute"`
	Burst             int      `yaml:"burst"`
	WhitelistIPs      []string `yaml:"whitelist_ips"`
	BlacklistIPs      []string `yaml:"blacklist_ips"`
	TrustedProxies    []string `yaml:"trusted_proxies"` // IPs of reverse proxies we trust
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
	if port := os.Getenv("LOGPULSE_PORT"); port != "" {
		cfg.Server.Port = port
	}
	if apiKey := os.Getenv("LOGPULSE_API_KEY"); apiKey != "" {
		cfg.Auth.APIKey = apiKey
		cfg.Auth.Enabled = true
	}
	if storagePath := os.Getenv("LOGPULSE_STORAGE_PATH"); storagePath != "" {
		cfg.Storage.Path = storagePath
	}

	// Rate limit environment variable overrides
	if enabled := os.Getenv("LOGPULSE_RATE_LIMIT_ENABLED"); enabled != "" {
		cfg.RateLimit.Enabled = enabled == "true"
	}
	if rpm := os.Getenv("LOGPULSE_RATE_LIMIT_RPM"); rpm != "" {
		if val, err := strconv.Atoi(rpm); err == nil {
			cfg.RateLimit.RequestsPerMinute = val
		}
	}
	if burst := os.Getenv("LOGPULSE_RATE_LIMIT_BURST"); burst != "" {
		if val, err := strconv.Atoi(burst); err == nil {
			cfg.RateLimit.Burst = val
		}
	}
	if whitelist := os.Getenv("LOGPULSE_RATE_LIMIT_WHITELIST"); whitelist != "" {
		cfg.RateLimit.WhitelistIPs = strings.Split(whitelist, ",")
	}
	if blacklist := os.Getenv("LOGPULSE_RATE_LIMIT_BLACKLIST"); blacklist != "" {
		cfg.RateLimit.BlacklistIPs = strings.Split(blacklist, ",")
	}
	if proxies := os.Getenv("LOGPULSE_RATE_LIMIT_TRUSTED_PROXIES"); proxies != "" {
		cfg.RateLimit.TrustedProxies = strings.Split(proxies, ",")
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
		RateLimit: RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 1000,
			Burst:             100,
			WhitelistIPs:      []string{},
			BlacklistIPs:      []string{},
			TrustedProxies:    []string{},
		},
	}
}
