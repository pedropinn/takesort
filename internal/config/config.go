package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	defaultWatchDir = "/media/temp"
	defaultMediaDir = "/media"

	defaultDebounceInterval = 2 * time.Second
	defaultLogLevel         = "info"
)

var validLogLevels = map[string]bool{
	"debug": true,
	"info":  true,
	"warn":  true,
	"error": true,
}

// Config holds application settings loaded from environment variables.
type Config struct {
	WatchDir     string
	MediaDir     string
	ConflictsDir string
	ErrorsDir    string

	DebounceInterval time.Duration
	LogLevel         string
}

// Load reads configuration from environment variables and applies defaults.
func Load() (*Config, error) {
	cfg := &Config{
		WatchDir:         defaultWatchDir,
		MediaDir:         defaultMediaDir,
		DebounceInterval: defaultDebounceInterval,
		LogLevel:         defaultLogLevel,
	}

	if v := os.Getenv("TAKESORT_WATCH_DIR"); v != "" {
		cfg.WatchDir = v
	}

	if v := os.Getenv("TAKESORT_MEDIA_DIR"); v != "" {
		cfg.MediaDir = v
	}

	cfg.ConflictsDir = filepath.Join(cfg.WatchDir, "conflicts")
	cfg.ErrorsDir = filepath.Join(cfg.WatchDir, "errors")

	if v := os.Getenv("TAKESORT_DEBOUNCE_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("invalid TAKESORT_DEBOUNCE_INTERVAL %q: %w", v, err)
		}
		if d <= 0 {
			return nil, fmt.Errorf("invalid TAKESORT_DEBOUNCE_INTERVAL %q: must be positive", v)
		}
		cfg.DebounceInterval = d
	}

	if v := os.Getenv("TAKESORT_LOG_LEVEL"); v != "" {
		if !validLogLevels[v] {
			return nil, fmt.Errorf("invalid TAKESORT_LOG_LEVEL %q: must be one of debug, info, warn, error", v)
		}
		cfg.LogLevel = v
	}

	return cfg, nil
}
