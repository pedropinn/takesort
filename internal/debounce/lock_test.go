package debounce

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHelperProcess is re-exec'd as a subprocess to hold a POSIX lock.
// POSIX fcntl locks are per-process, so a separate process is required.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_HELPER_LOCK_FILE") == "" {
		return
	}
	path := os.Getenv("GO_HELPER_LOCK_FILE")

	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = f.Close() }()

	lock := syscall.Flock_t{
		Type:   syscall.F_WRLCK,
		Whence: 0,
		Start:  0,
		Len:    0,
	}
	if err := syscall.FcntlFlock(f.Fd(), syscall.F_SETLK, &lock); err != nil {
		fmt.Fprintf(os.Stderr, "fcntl: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("locked")

	// Block until parent kills this process
	select {}
}

func TestIsFileLocked_UnlockedFileReturnsFalse(t *testing.T) {
	f := filepath.Join(t.TempDir(), "unlocked.mp4")
	require.NoError(t, os.WriteFile(f, []byte("data"), 0o644))

	assert.False(t, isFileLocked(f))
}

func TestIsFileLocked_LockedByAnotherProcessReturnsTrue(t *testing.T) {
	f := filepath.Join(t.TempDir(), "locked.mp4")
	require.NoError(t, os.WriteFile(f, []byte("data"), 0o644))

	// Spawn subprocess that holds a POSIX write lock
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
	cmd.Env = append(os.Environ(), "GO_HELPER_LOCK_FILE="+f)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	// Wait for subprocess to signal it holds the lock
	scanner := bufio.NewScanner(stdout)
	require.True(t, scanner.Scan(), "subprocess did not emit 'locked' signal")
	require.Equal(t, "locked", scanner.Text())

	assert.True(t, isFileLocked(f))
}

func TestIsFileLocked_AfterUnlockReturnsFalse(t *testing.T) {
	f := filepath.Join(t.TempDir(), "relocked.mp4")
	require.NoError(t, os.WriteFile(f, []byte("data"), 0o644))

	// Spawn and immediately kill subprocess so lock is released
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
	cmd.Env = append(os.Environ(), "GO_HELPER_LOCK_FILE="+f)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())

	scanner := bufio.NewScanner(stdout)
	require.True(t, scanner.Scan())

	// Kill subprocess to release the lock
	require.NoError(t, cmd.Process.Kill())
	_ = cmd.Wait()

	assert.False(t, isFileLocked(f))
}

func TestIsFileLocked_NonExistentFileReturnsFalse(t *testing.T) {
	assert.False(t, isFileLocked("/tmp/nonexistent_file_takesort_test.mp4"))
}

func TestIsFileLocked_ReadOnlyFileReturnsFalse(t *testing.T) {
	f := filepath.Join(t.TempDir(), "readonly.mp4")
	require.NoError(t, os.WriteFile(f, []byte("data"), 0o444))

	assert.False(t, isFileLocked(f))
}
