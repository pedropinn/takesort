package safepath

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSymlink_RegularFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(f, []byte("data"), 0o600))

	assert.False(t, IsSymlink(f))
}

func TestIsSymlink_Symlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks not reliable on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	require.NoError(t, os.WriteFile(target, []byte("data"), 0o600))

	link := filepath.Join(dir, "link.txt")
	require.NoError(t, os.Symlink(target, link))

	assert.True(t, IsSymlink(link))
}

func TestIsSymlink_NonExistent(t *testing.T) {
	assert.False(t, IsSymlink("/nonexistent/path"))
}

func TestValidateFile_RegularFileInsideBase(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(f, []byte("data"), 0o600))

	assert.NoError(t, ValidateFile(f, dir))
}

func TestValidateFile_Symlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks not reliable on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	require.NoError(t, os.WriteFile(target, []byte("data"), 0o600))

	link := filepath.Join(dir, "link.txt")
	require.NoError(t, os.Symlink(target, link))

	err := ValidateFile(link, dir)
	assert.ErrorIs(t, err, ErrSymlink)
}

func TestValidateFile_OutsideBase(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	f := filepath.Join(dir1, "file.txt")
	require.NoError(t, os.WriteFile(f, []byte("data"), 0o600))

	err := ValidateFile(f, dir2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "outside allowed directory")
}
