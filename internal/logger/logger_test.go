package logger_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/pinn/takesort/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetup_InfoLevelDoesNotEmitDebug(t *testing.T) {
	var buf bytes.Buffer
	log := logger.SetupWithWriter("info", &buf)

	log.Debug("this should not appear")
	log.Info("this should appear")

	output := buf.String()
	assert.NotContains(t, output, "this should not appear")
	assert.Contains(t, output, "this should appear")
}

func TestSetup_EmitsValidJSON(t *testing.T) {
	var buf bytes.Buffer
	log := logger.SetupWithWriter("info", &buf)

	log.Info("test message", slog.String("file", "example.mp4"))

	var entry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Contains(t, entry, "time")
	assert.Contains(t, entry, "level")
	assert.Contains(t, entry, "msg")
	assert.Equal(t, "test message", entry["msg"])
}
