package debounce_test

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/pinn/takesort/internal/debounce"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaitForStability_StableFileReturnsNil(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "stable.mp4")
	require.NoError(t, os.WriteFile(f, []byte("video content"), 0o644))

	err := debounce.WaitForStability(context.Background(), f, 50*time.Millisecond, 4)

	assert.NoError(t, err)
}

func TestWaitForStability_ChangingSizeResetsCounter(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "growing.mp4")
	require.NoError(t, os.WriteFile(f, []byte("a"), 0o644))

	done := make(chan error, 1)
	go func() {
		done <- debounce.WaitForStability(context.Background(), f, 50*time.Millisecond, 4)
	}()

	// Grow the file after two checks (~100ms)
	time.Sleep(110 * time.Millisecond)
	require.NoError(t, os.WriteFile(f, []byte("ab"), 0o644))

	err := <-done
	assert.NoError(t, err)
}

func TestWaitForStability_FileDisappearedReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "vanish.mp4")
	require.NoError(t, os.WriteFile(f, []byte("data"), 0o644))

	done := make(chan error, 1)
	go func() {
		done <- debounce.WaitForStability(context.Background(), f, 50*time.Millisecond, 4)
	}()

	// Remove the file after a short delay
	time.Sleep(80 * time.Millisecond)
	require.NoError(t, os.Remove(f))

	err := <-done
	assert.ErrorIs(t, err, debounce.ErrFileDisappeared)
}

func TestWaitForStability_ConfigurableInterval(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "timing.mp4")
	require.NoError(t, os.WriteFile(f, []byte("content"), 0o644))

	start := time.Now()
	err := debounce.WaitForStability(context.Background(), f, 30*time.Millisecond, 4)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(110))
}

func TestWaitForStability_LockedFileResetsCounter(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "locked.mp4")
	require.NoError(t, os.WriteFile(f, []byte("video data"), 0o644))

	// Spawn subprocess to hold a POSIX write lock (fcntl locks are per-process)
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
	cmd.Env = append(os.Environ(), "GO_HELPER_LOCK_FILE="+f)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())

	// Wait for subprocess to acquire lock
	scanner := bufio.NewScanner(stdout)
	require.True(t, scanner.Scan(), "subprocess did not emit 'locked' signal")

	done := make(chan error, 1)
	go func() {
		done <- debounce.WaitForStability(context.Background(), f, 50*time.Millisecond, 3)
	}()

	// Release lock after 200ms by killing subprocess
	time.Sleep(200 * time.Millisecond)
	require.NoError(t, cmd.Process.Kill())
	_ = cmd.Wait()

	err = <-done
	assert.NoError(t, err)
}

func TestWaitForStability_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "cancel.mp4")
	require.NoError(t, os.WriteFile(f, []byte("data"), 0o644))

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- debounce.WaitForStability(ctx, f, 5*time.Second, 10)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	err := <-done
	assert.ErrorIs(t, err, context.Canceled)
}
