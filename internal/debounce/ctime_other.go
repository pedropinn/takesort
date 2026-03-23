//go:build !linux

package debounce

import (
	"os"
	"time"
)

// extractCtime is a no-op on non-Linux platforms where syscall.Stat_t is
// unavailable. Stability checks fall back to size-only detection.
func extractCtime(_ os.FileInfo) time.Time {
	return time.Time{}
}
