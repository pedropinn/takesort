package config_test

import (
	"testing"
	"time"

	"github.com/pinn/takesort/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_DefaultValues(t *testing.T) {
	t.Setenv("TAKESORT_DEBOUNCE_INTERVAL", "")
	t.Setenv("TAKESORT_LOG_LEVEL", "")
	t.Setenv("TAKESORT_WATCH_DIR", "")
	t.Setenv("TAKESORT_MEDIA_DIR", "")

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, "/media/temp", cfg.WatchDir)
	assert.Equal(t, "/media", cfg.MediaDir)
	assert.Equal(t, "/media/temp/conflicts", cfg.ConflictsDir)
	assert.Equal(t, "/media/temp/errors", cfg.ErrorsDir)
	assert.Equal(t, 2*time.Second, cfg.DebounceInterval)
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestLoad_CustomPaths(t *testing.T) {
	t.Setenv("TAKESORT_WATCH_DIR", "/media/temp")
	t.Setenv("TAKESORT_MEDIA_DIR", "/media")

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, "/media/temp", cfg.WatchDir)
	assert.Equal(t, "/media", cfg.MediaDir)
	assert.Equal(t, "/media/temp/conflicts", cfg.ConflictsDir)
	assert.Equal(t, "/media/temp/errors", cfg.ErrorsDir)
}

func TestLoad_CustomEnvValues(t *testing.T) {
	t.Setenv("TAKESORT_DEBOUNCE_INTERVAL", "5s")
	t.Setenv("TAKESORT_LOG_LEVEL", "debug")

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, 5*time.Second, cfg.DebounceInterval)
	assert.Equal(t, "debug", cfg.LogLevel)
}

func TestLoad_InvalidDebounceInterval(t *testing.T) {
	t.Setenv("TAKESORT_DEBOUNCE_INTERVAL", "abc")

	_, err := config.Load()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TAKESORT_DEBOUNCE_INTERVAL")
}
