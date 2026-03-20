package conflict_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pinn/takesort/internal/conflict"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasConflict_FileWithoutConflictMovedNormally(t *testing.T) {
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "clip.mp4")

	result := conflict.HasConflict(destPath)

	assert.False(t, result)
}

func TestMoveToConflicts_DuplicateFilenameRedirectedToConflicts(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "temp")
	conflictsDir := filepath.Join(tmpDir, "temp", "conflicts")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.MkdirAll(conflictsDir, 0o755))

	srcFile := filepath.Join(srcDir, "clip.mp4")
	require.NoError(t, os.WriteFile(srcFile, []byte("duplicate"), 0o644))

	err := conflict.MoveToConflicts(srcFile, conflictsDir)

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(conflictsDir, "clip.mp4"))
	assert.NoFileExists(t, srcFile)
}

func TestMoveToConflicts_ConflictsDirCreatedAutomatically(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "temp")
	conflictsDir := filepath.Join(tmpDir, "temp", "conflicts")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))

	srcFile := filepath.Join(srcDir, "clip.mp4")
	require.NoError(t, os.WriteFile(srcFile, []byte("duplicate"), 0o644))

	// conflictsDir does not exist yet
	assert.NoDirExists(t, conflictsDir)

	err := conflict.MoveToConflicts(srcFile, conflictsDir)

	require.NoError(t, err)
	assert.DirExists(t, conflictsDir)
	assert.FileExists(t, filepath.Join(conflictsDir, "clip.mp4"))
}
