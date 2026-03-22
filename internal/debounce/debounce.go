package debounce

import (
	"context"
	"errors"
	"os"
	"time"
)

// ErrFileDisappeared is returned when a file is removed during stability checks.
var ErrFileDisappeared = errors.New("file disappeared during stability check")

// WaitForStability polls file size and inode change time (ctime) at the given
// interval and returns nil once both remain unchanged for the specified number
// of consecutive checks. The first read establishes a baseline and does not
// count as a stable check.
//
// The ctime check detects files still being written via SMB: when a client
// copies a file over SMB, the server may pre-allocate the file at its final
// size (so Size never changes), and SMB clients control mtime (setting it to
// the original file's timestamp). However, every write() by smbd causes the
// kernel to update the inode's ctime. Once the copy finishes and the file
// handle is closed, ctime stabilises.
func WaitForStability(ctx context.Context, path string, interval time.Duration, checks int) error {
	prevSize := int64(-1)
	prevCtime := time.Time{}
	stable := 0

	for stable < checks {
		info, err := os.Lstat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return ErrFileDisappeared
			}
			return err
		}

		size := info.Size()
		ctime := extractCtime(info)
		if size == prevSize && ctime.Equal(prevCtime) {
			stable++
		} else {
			stable = 0
			prevSize = size
			prevCtime = ctime
		}

		if stable >= checks {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}

	return nil
}
