package debounce

import (
	"os"
	"syscall"
	"time"
)

// extractCtime returns the inode change time (ctime) from the file info.
// Unlike mtime which SMB clients can set to the original file's timestamp,
// ctime is updated by the kernel on every write() or metadata change. This
// makes it reliable for detecting files still being written via SMB: even
// when the file is pre-allocated at full size (so Size never changes), smbd's
// ongoing writes keep bumping ctime until the copy finishes and the file
// handle is closed.
func extractCtime(info os.FileInfo) time.Time {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return time.Time{}
	}
	return time.Unix(stat.Ctim.Sec, stat.Ctim.Nsec)
}
