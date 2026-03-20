package watcher_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pinn/takesort/internal/watcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWatcher_CreateEventEmitsFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	w := watcher.New(tmpDir, nil, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = w.Start(ctx)
	}()

	// Give the watcher time to start
	time.Sleep(100 * time.Millisecond)

	f := filepath.Join(tmpDir, "clip.mp4")
	require.NoError(t, os.WriteFile(f, []byte("video"), 0o644))

	select {
	case path := <-w.Events():
		assert.Equal(t, f, path)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestWatcher_EventsInIgnoredSubdirsAreSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	conflictsDir := filepath.Join(tmpDir, "conflicts")
	errorsDir := filepath.Join(tmpDir, "errors")
	require.NoError(t, os.MkdirAll(conflictsDir, 0o755))
	require.NoError(t, os.MkdirAll(errorsDir, 0o755))

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	w := watcher.New(tmpDir, []string{conflictsDir, errorsDir}, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = w.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Create files in ignored subdirectories
	require.NoError(t, os.WriteFile(filepath.Join(conflictsDir, "dup.mp4"), []byte("dup"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(errorsDir, "bad.mp4"), []byte("bad"), 0o644))

	// Create a valid file in the watch root
	validFile := filepath.Join(tmpDir, "valid.mp4")
	time.Sleep(100 * time.Millisecond)
	require.NoError(t, os.WriteFile(validFile, []byte("ok"), 0o644))

	select {
	case path := <-w.Events():
		assert.Equal(t, validFile, path)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for valid event")
	}
}

func TestWatcher_StopsOnContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	w := watcher.New(tmpDir, nil, log)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- w.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not stop after context cancellation")
	}
}
