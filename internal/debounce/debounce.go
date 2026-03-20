package debounce

import (
	"errors"
	"os"
	"time"
)

// ErrFileDisappeared is returned when a file is removed during stability checks.
var ErrFileDisappeared = errors.New("file disappeared during stability check")

// WaitForStability polls the file size at the given interval and returns nil
// once the size remains unchanged for the specified number of consecutive checks.
func WaitForStability(path string, interval time.Duration, checks int) error {
	prevSize := int64(-1)
	stable := 0

	for stable < checks {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return ErrFileDisappeared
			}
			return err
		}

		size := info.Size()
		if size == prevSize {
			stable++
		} else {
			stable = 1
			prevSize = size
		}

		if stable >= checks {
			return nil
		}

		time.Sleep(interval)
	}

	return nil
}
