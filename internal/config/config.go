// Package config holds runtime connection settings for the EdgeX microservices.
package config

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Config points directly at the EdgeX microservices (no edgex-ui-go proxy).
type Config struct {
	MetadataURL string  `json:"metadataUrl"`
	DataURL     string  `json:"dataUrl"`
	CommandURL  string  `json:"commandUrl"`
	MappingsURL string  `json:"mappingsUrl"`
	KeeperURL   string  `json:"keeperUrl"`
	SystemURL   string  `json:"systemUrl"`
	Token       string  `json:"token,omitempty"`
	TimeoutSec  float64 `json:"timeoutSec"`
}

// Timeout returns the per-request timeout.
func (c Config) Timeout() time.Duration {
	seconds := c.TimeoutSec
	if seconds <= 0 {
		seconds = 8
	}
	return time.Duration(seconds * float64(time.Second))
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

// Load reads configuration from the environment with local defaults.
func Load() Config {
	timeout := 8.0
	if raw := strings.TrimSpace(os.Getenv("EDGEX_TIMEOUT")); raw != "" {
		if parsed, err := strconv.ParseFloat(raw, 64); err == nil && parsed > 0 {
			timeout = parsed
		}
	}
	return Config{
		MetadataURL: env("EDGEX_METADATA_URL", "http://localhost:59881"),
		DataURL:     env("EDGEX_DATA_URL", "http://localhost:59880"),
		CommandURL:  env("EDGEX_COMMAND_URL", "http://localhost:59882"),
		MappingsURL: env("EDGEX_MAPPINGS_URL", "http://localhost:59883"),
		KeeperURL:   env("EDGEX_KEEPER_URL", "http://localhost:59890"),
		SystemURL:   env("EDGEX_SYSTEM_URL", "http://localhost:59886"),
		Token:       os.Getenv("EDGEX_TOKEN"),
		TimeoutSec:  timeout,
	}
}

// Store keeps the active configuration so the UI can update it at runtime.
type Store struct {
	mu  sync.RWMutex
	cfg Config
}

// NewStore wraps an initial configuration.
func NewStore(cfg Config) *Store {
	return &Store{cfg: cfg}
}

// Get returns a copy of the current configuration.
func (s *Store) Get() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

// Set replaces the current configuration.
func (s *Store) Set(cfg Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
}
