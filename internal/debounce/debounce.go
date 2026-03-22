package debounce

import (
	"context"
	"errors"
	"os"
	"time"
)

// ErrFileDisappeared is returned when a file is removed during stability checks.
var ErrFileDisappeared = errors.New("file disappeared during stability check")

// WaitForStability polls file size and checks for POSIX locks at the given
// interval and returns nil once the size remains unchanged AND the file is not
// locked for the specified number of consecutive checks. The first read
// establishes a baseline and does not count as a stable check.
//
// The lock check detects files still being written via SMB/NFS: when a client
// copies a file over SMB, the Samba server holds a POSIX (fcntl) lock on the
// file. Even if the filesystem pre-allocates the file at full size, the lock
// check will catch that the copy is still in progress.
func WaitForStability(ctx context.Context, path string, interval time.Duration, checks int) error {
	prevSize := int64(-1)
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
		if size == prevSize && !isFileLocked(path) {
			stable++
		} else {
			stable = 0
			prevSize = size
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
