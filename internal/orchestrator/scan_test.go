package orchestrator_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/pinn/takesort/internal/orchestrator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanExisting_FilesInTempAreReturned(t *testing.T) {
	tmpDir := t.TempDir()
	watchDir := filepath.Join(tmpDir, "temp")
	require.NoError(t, os.MkdirAll(watchDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(watchDir, "clip.mp4"), []byte("video"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(watchDir, "photo.jpg"), []byte("jpeg"), 0o644))

	files, err := orchestrator.ScanExisting(watchDir, nil)

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

	files, err := orchestrator.ScanExisting(watchDir, []string{conflictsDir, errorsDir})

	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, filepath.Join(watchDir, "orphan.mp4"), files[0])
}

func TestScanExisting_EmptyDirReturnsNoError(t *testing.T) {
	tmpDir := t.TempDir()
	watchDir := filepath.Join(tmpDir, "temp")
	require.NoError(t, os.MkdirAll(watchDir, 0o755))

	files, err := orchestrator.ScanExisting(watchDir, nil)

	require.NoError(t, err)
	assert.Empty(t, files)
}
