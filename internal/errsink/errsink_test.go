package errsink_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pinn/takesort/internal/errsink"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMoveToErrors_FileWithErrorMovedToErrors(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "temp")
	errorsDir := filepath.Join(tmpDir, "temp", "errors")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.MkdirAll(errorsDir, 0o755))

	srcFile := filepath.Join(srcDir, "broken.mp4")
	require.NoError(t, os.WriteFile(srcFile, []byte("bad data"), 0o644))

	err := errsink.MoveToErrors(srcFile, errorsDir)

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(errorsDir, "broken.mp4"))
	assert.NoFileExists(t, srcFile)
}

func TestMoveToErrors_ErrorsDirCreatedAutomatically(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "temp")
	errorsDir := filepath.Join(tmpDir, "temp", "errors")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))

	srcFile := filepath.Join(srcDir, "broken.mp4")
	require.NoError(t, os.WriteFile(srcFile, []byte("bad data"), 0o644))

	// errorsDir does not exist yet
	assert.NoDirExists(t, errorsDir)

	err := errsink.MoveToErrors(srcFile, errorsDir)

	require.NoError(t, err)
	assert.DirExists(t, errorsDir)
	assert.FileExists(t, filepath.Join(errorsDir, "broken.mp4"))
}

func TestShouldRejectFile_UnknownExtensionMovedToErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// zero-size file
	zeroFile := filepath.Join(tmpDir, "empty.mp4")
	require.NoError(t, os.WriteFile(zeroFile, []byte{}, 0o644))

	reject, reason := errsink.ShouldRejectFile(zeroFile)
	assert.True(t, reject)
	assert.Equal(t, "zero size file", reason)

	// file without extension
	noExtFile := filepath.Join(tmpDir, "noext")
	require.NoError(t, os.WriteFile(noExtFile, []byte("content"), 0o644))

	reject, reason = errsink.ShouldRejectFile(noExtFile)
	assert.True(t, reject)
	assert.Equal(t, "file without extension", reason)

	// valid file
	validFile := filepath.Join(tmpDir, "video.mp4")
	require.NoError(t, os.WriteFile(validFile, []byte("video data"), 0o644))

	reject, reason = errsink.ShouldRejectFile(validFile)
	assert.False(t, reject)
	assert.Empty(t, reason)
}
