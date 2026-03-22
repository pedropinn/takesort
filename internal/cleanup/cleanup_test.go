package cleanup_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pinn/takesort/internal/cleanup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanEmptyDirs_RemovesEmptyNestedDirs(t *testing.T) {
	root := t.TempDir()
	// Create a nested empty directory tree: root/a/b/c (all empty)
	deepDir := filepath.Join(root, "a", "b", "c")
	require.NoError(t, os.MkdirAll(deepDir, 0o755))

	err := cleanup.CleanEmptyDirs(root, nil)
	require.NoError(t, err)

	assert.NoDirExists(t, filepath.Join(root, "a"))
	assert.DirExists(t, root)
}

func TestCleanEmptyDirs_PreservesNonEmptyDirs(t *testing.T) {
	root := t.TempDir()
	// Create a/b/c where b contains a file
	dirWithFile := filepath.Join(root, "a", "b")
	emptyChild := filepath.Join(root, "a", "b", "c")
	require.NoError(t, os.MkdirAll(emptyChild, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dirWithFile, "keep.txt"), []byte("data"), 0o644))

	err := cleanup.CleanEmptyDirs(root, nil)
	require.NoError(t, err)

	// c was empty so it's removed, but b has a file so it stays
	assert.NoDirExists(t, emptyChild)
	assert.DirExists(t, dirWithFile)
	assert.FileExists(t, filepath.Join(dirWithFile, "keep.txt"))
}

func TestCleanEmptyDirs_SkipsIgnoredDirs(t *testing.T) {
	root := t.TempDir()
	conflictsDir := filepath.Join(root, "conflicts")
	errorsDir := filepath.Join(root, "errors")
	require.NoError(t, os.MkdirAll(conflictsDir, 0o755))
	require.NoError(t, os.MkdirAll(errorsDir, 0o755))

	err := cleanup.CleanEmptyDirs(root, []string{conflictsDir, errorsDir})
	require.NoError(t, err)

	// Ignored dirs must remain even though they are empty
	assert.DirExists(t, conflictsDir)
	assert.DirExists(t, errorsDir)
}

func TestCleanEmptyDirs_NeverRemovesRoot(t *testing.T) {
	root := t.TempDir()
	// Root is empty (no subdirs, no files) -- it must still survive
	err := cleanup.CleanEmptyDirs(root, nil)
	require.NoError(t, err)

	assert.DirExists(t, root)
}
