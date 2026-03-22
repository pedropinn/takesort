//go:build !unix

package debounce

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsFileLocked_AlwaysReturnsFalseOnNonUnix(t *testing.T) {
	f := filepath.Join(t.TempDir(), "file.mp4")
	require.NoError(t, os.WriteFile(f, []byte("data"), 0o644))

	assert.False(t, isFileLocked(f))
}

func TestIsFileLocked_NonExistentFileReturnsFalseOnNonUnix(t *testing.T) {
	assert.False(t, isFileLocked("/tmp/nonexistent_takesort.mp4"))
}
