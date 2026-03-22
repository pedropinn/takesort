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

func TestWatcher_ExistingSubdirFilesAreEmitted(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "DCIM", "100MEDIA")
	require.NoError(t, os.MkdirAll(subDir, 0o755))

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	w := watcher.New(tmpDir, nil, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = w.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Create a file in a pre-existing subdirectory
	f := filepath.Join(subDir, "DJI_0001.mp4")
	require.NoError(t, os.WriteFile(f, []byte("drone video"), 0o644))

	select {
	case path := <-w.Events():
		assert.Equal(t, f, path)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event from subdirectory")
	}
}

func TestWatcher_NewSubdirDynamicallyWatched(t *testing.T) {
	tmpDir := t.TempDir()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	w := watcher.New(tmpDir, nil, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = w.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Create a new subdirectory at runtime
	newDir := filepath.Join(tmpDir, "newdir")
	require.NoError(t, os.MkdirAll(newDir, 0o755))

	time.Sleep(200 * time.Millisecond)

	// Create a file in the newly created subdirectory
	f := filepath.Join(newDir, "clip.mp4")
	require.NoError(t, os.WriteFile(f, []byte("video"), 0o644))

	select {
	case path := <-w.Events():
		assert.Equal(t, f, path)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event from dynamically watched subdirectory")
	}
}

func TestWatcher_HiddenDirCreatedAtRuntimeIsDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	w := watcher.New(tmpDir, nil, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = w.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Create a hidden directory at runtime
	hiddenDir := filepath.Join(tmpDir, ".Trashes")
	require.NoError(t, os.MkdirAll(hiddenDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(hiddenDir, "junk.dat"), []byte("junk"), 0o644))

	time.Sleep(300 * time.Millisecond)

	// Hidden dir should have been deleted
	assert.NoDirExists(t, hiddenDir)

	// Create a regular file to prove the watcher is still running
	validFile := filepath.Join(tmpDir, "valid.mp4")
	require.NoError(t, os.WriteFile(validFile, []byte("ok"), 0o644))

	select {
	case path := <-w.Events():
		assert.Equal(t, validFile, path)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for valid event after hidden dir deletion")
	}
}
