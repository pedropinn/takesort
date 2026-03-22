//go:build unix

package debounce

import (
	"os"
	"syscall"
)

// isFileLocked tries to acquire a POSIX write lock (fcntl F_SETLK) on the
// file. If another process (e.g. smbd during an SMB copy) holds a lock, the
// attempt fails and this returns true — meaning the file is still in use.
func isFileLocked(path string) bool {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return false
	}
	defer f.Close()

	lock := syscall.Flock_t{
		Type:   syscall.F_WRLCK,
		Whence: 0,
		Start:  0,
		Len:    0,
	}
	err = syscall.FcntlFlock(f.Fd(), syscall.F_SETLK, &lock)
	if err != nil {
		return true
	}

	lock.Type = syscall.F_UNLCK
	_ = syscall.FcntlFlock(f.Fd(), syscall.F_SETLK, &lock)
	return false
}
