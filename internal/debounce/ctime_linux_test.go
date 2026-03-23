//go:build linux

package debounce_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pinn/takesort/internal/debounce"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaitForStability_ChangingCtimeResetsCounter(t *testing.T) {
	f := filepath.Join(t.TempDir(), "ctime.mp4")
	require.NoError(t, os.WriteFile(f, []byte("video data"), 0o644))

	done := make(chan error, 1)
	go func() {
		done <- debounce.WaitForStability(context.Background(), f, 50*time.Millisecond, 4)
	}()

	// After ~110ms (2 checks done), change ctime without changing size.
	// chmod updates ctime only, which should reset the stability counter.
	time.Sleep(110 * time.Millisecond)
	require.NoError(t, os.Chmod(f, 0o600))

	start := time.Now()
	err := <-done
	elapsed := time.Since(start)

	assert.NoError(t, err)
	// After the chmod at ~110ms, 4 more stable checks needed (4×50ms = 200ms).
	// So total elapsed from the chmod should be at least ~200ms.
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(150),
		"ctime change should have reset the stability counter")
}
