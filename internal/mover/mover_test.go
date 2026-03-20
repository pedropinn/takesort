package mover_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pinn/takesort/internal/classifier"
	"github.com/pinn/takesort/internal/mover"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDestPath_VideoMovedToDateRoot(t *testing.T) {
	modTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)

	dest := mover.BuildDestPath("/media", modTime, classifier.Video)

	assert.Equal(t, filepath.Join("/media", "2026", "2026-03-15"), dest)
}

func TestBuildDestPath_PhotoJPEGMovedToPhotosJpeg(t *testing.T) {
	modTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)

	dest := mover.BuildDestPath("/media", modTime, classifier.PhotoJPEG)

	assert.Equal(t, filepath.Join("/media", "2026", "2026-03-15", "photos", "jpeg"), dest)
}

func TestBuildDestPath_PhotoRAWMovedToPhotosRaw(t *testing.T) {
	modTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)

	dest := mover.BuildDestPath("/media", modTime, classifier.PhotoRAW)

	assert.Equal(t, filepath.Join("/media", "2026", "2026-03-15", "photos", "raw"), dest)
}

func TestBuildDestPath_AudioMovedToAudioDir(t *testing.T) {
	modTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)

	dest := mover.BuildDestPath("/media", modTime, classifier.Audio)

	assert.Equal(t, filepath.Join("/media", "2026", "2026-03-15", "audio"), dest)
}

func TestBuildDestPath_ProxyMovedToProxyDir(t *testing.T) {
	modTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)

	dest := mover.BuildDestPath("/media", modTime, classifier.Proxy)

	assert.Equal(t, filepath.Join("/media", "2026", "2026-03-15", "proxy"), dest)
}

func TestMoveFile_VideoMovedToDateDir(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "temp")
	mediaDir := filepath.Join(tmpDir, "media")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))

	srcFile := filepath.Join(srcDir, "clip.mp4")
	require.NoError(t, os.WriteFile(srcFile, []byte("video content"), 0o644))

	modTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	destDir := mover.BuildDestPath(mediaDir, modTime, classifier.Video)

	err := mover.MoveFile(srcFile, destDir)

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(destDir, "clip.mp4"))
	assert.NoFileExists(t, srcFile)
}

func TestMoveFile_PhotoJPEGMovedToPhotosJpeg(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "temp")
	mediaDir := filepath.Join(tmpDir, "media")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))

	srcFile := filepath.Join(srcDir, "photo.jpg")
	require.NoError(t, os.WriteFile(srcFile, []byte("jpeg data"), 0o644))

	modTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	destDir := mover.BuildDestPath(mediaDir, modTime, classifier.PhotoJPEG)

	err := mover.MoveFile(srcFile, destDir)

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(destDir, "photo.jpg"))
	assert.NoFileExists(t, srcFile)
}

func TestMoveFile_PhotoRAWMovedToPhotosRaw(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "temp")
	mediaDir := filepath.Join(tmpDir, "media")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))

	srcFile := filepath.Join(srcDir, "photo.dng")
	require.NoError(t, os.WriteFile(srcFile, []byte("raw data"), 0o644))

	modTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	destDir := mover.BuildDestPath(mediaDir, modTime, classifier.PhotoRAW)

	err := mover.MoveFile(srcFile, destDir)

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(destDir, "photo.dng"))
	assert.NoFileExists(t, srcFile)
}

func TestMoveFile_AudioMovedToAudioDir(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "temp")
	mediaDir := filepath.Join(tmpDir, "media")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))

	srcFile := filepath.Join(srcDir, "sound.wav")
	require.NoError(t, os.WriteFile(srcFile, []byte("wav data"), 0o644))

	modTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	destDir := mover.BuildDestPath(mediaDir, modTime, classifier.Audio)

	err := mover.MoveFile(srcFile, destDir)

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(destDir, "sound.wav"))
	assert.NoFileExists(t, srcFile)
}

func TestMoveFile_CreatesDirectoryAutomatically(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "temp")
	mediaDir := filepath.Join(tmpDir, "media")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))

	srcFile := filepath.Join(srcDir, "clip.mp4")
	require.NoError(t, os.WriteFile(srcFile, []byte("video content"), 0o644))

	destDir := filepath.Join(mediaDir, "2026", "2026-03-15")

	// destDir does not exist yet
	assert.NoDirExists(t, destDir)

	err := mover.MoveFile(srcFile, destDir)

	require.NoError(t, err)
	assert.DirExists(t, destDir)
	assert.FileExists(t, filepath.Join(destDir, "clip.mp4"))
}
