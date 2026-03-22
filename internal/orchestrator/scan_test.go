package orchestrator_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/pinn/takesort/internal/orchestrator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testLogger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func TestScanExisting_FilesInTempAreReturned(t *testing.T) {
	tmpDir := t.TempDir()
	watchDir := filepath.Join(tmpDir, "temp")
	require.NoError(t, os.MkdirAll(watchDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(watchDir, "clip.mp4"), []byte("video"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(watchDir, "photo.jpg"), []byte("jpeg"), 0o644))

	files, err := orchestrator.ScanExisting(watchDir, nil, testLogger)

	require.NoError(t, err)
	sort.Strings(files)
	assert.Len(t, files, 2)
	assert.Equal(t, filepath.Join(watchDir, "clip.mp4"), files[0])
	assert.Equal(t, filepath.Join(watchDir, "photo.jpg"), files[1])
}

func TestScanExisting_FilesInIgnoredSubdirsAreExcluded(t *testing.T) {
	tmpDir := t.TempDir()
	watchDir := filepath.Join(tmpDir, "temp")
	conflictsDir := filepath.Join(watchDir, "conflicts")
	errorsDir := filepath.Join(watchDir, "errors")
	require.NoError(t, os.MkdirAll(conflictsDir, 0o755))
	require.NoError(t, os.MkdirAll(errorsDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(watchDir, "orphan.mp4"), []byte("video"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(conflictsDir, "dup.mp4"), []byte("dup"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(errorsDir, "bad.mp4"), []byte("bad"), 0o644))

	files, err := orchestrator.ScanExisting(watchDir, []string{conflictsDir, errorsDir}, testLogger)

	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, filepath.Join(watchDir, "orphan.mp4"), files[0])
}

func TestScanExisting_EmptyDirReturnsNoError(t *testing.T) {
	tmpDir := t.TempDir()
	watchDir := filepath.Join(tmpDir, "temp")
	require.NoError(t, os.MkdirAll(watchDir, 0o755))

	files, err := orchestrator.ScanExisting(watchDir, nil, testLogger)

	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestScanExisting_ReturnsFilesFromNestedSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()
	watchDir := filepath.Join(tmpDir, "temp")
	dcimDir := filepath.Join(watchDir, "DCIM", "100MEDIA")
	require.NoError(t, os.MkdirAll(dcimDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(watchDir, "root.mp4"), []byte("video"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dcimDir, "DJI_0001.mp4"), []byte("drone"), 0o644))

	files, err := orchestrator.ScanExisting(watchDir, nil, testLogger)

	require.NoError(t, err)
	sort.Strings(files)
	assert.Len(t, files, 2)
	assert.Contains(t, files, filepath.Join(watchDir, "root.mp4"))
	assert.Contains(t, files, filepath.Join(dcimDir, "DJI_0001.mp4"))
}

func TestScanExisting_DeletesHiddenDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	watchDir := filepath.Join(tmpDir, "temp")
	hiddenDir := filepath.Join(watchDir, ".Trashes")
	require.NoError(t, os.MkdirAll(hiddenDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(hiddenDir, "junk.dat"), []byte("junk"), 0o644))

	// Also place a regular file at root level
	require.NoError(t, os.WriteFile(filepath.Join(watchDir, "clip.mp4"), []byte("video"), 0o644))

	files, err := orchestrator.ScanExisting(watchDir, nil, testLogger)

	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, filepath.Join(watchDir, "clip.mp4"), files[0])
	assert.NoDirExists(t, hiddenDir)
}

func TestScanExisting_ExcludesIgnoredDirsAtAnyDepth(t *testing.T) {
	tmpDir := t.TempDir()
	watchDir := filepath.Join(tmpDir, "temp")
	conflictsDir := filepath.Join(watchDir, "conflicts")
	errorsDir := filepath.Join(watchDir, "errors")

	// Create nested content inside ignored dirs
	nestedConflict := filepath.Join(conflictsDir, "sub")
	require.NoError(t, os.MkdirAll(nestedConflict, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nestedConflict, "dup.mp4"), []byte("dup"), 0o644))

	nestedError := filepath.Join(errorsDir, "sub")
	require.NoError(t, os.MkdirAll(nestedError, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nestedError, "bad.mp4"), []byte("bad"), 0o644))

	// Normal subdir with a file
	normalSubdir := filepath.Join(watchDir, "DCIM")
	require.NoError(t, os.MkdirAll(normalSubdir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(normalSubdir, "photo.jpg"), []byte("pic"), 0o644))

	files, err := orchestrator.ScanExisting(watchDir, []string{conflictsDir, errorsDir}, testLogger)

	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, filepath.Join(normalSubdir, "photo.jpg"), files[0])
}
