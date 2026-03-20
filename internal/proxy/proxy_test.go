package proxy_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pinn/takesort/internal/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessProxy_LRFRenamedToMP4AndMovedToProxy(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "temp")
	proxyDir := filepath.Join(tmpDir, "media", "2026", "2026-03-15", "proxy")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))

	srcFile := filepath.Join(srcDir, "DJI_0001.lrf")
	require.NoError(t, os.WriteFile(srcFile, []byte("proxy data"), 0o644))

	err := proxy.ProcessProxy(srcFile, proxyDir)

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(proxyDir, "DJI_0001.mp4"))
	assert.NoFileExists(t, srcFile)
}

func TestProcessProxy_LRVRenamedToMP4AndMovedToProxy(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "temp")
	proxyDir := filepath.Join(tmpDir, "media", "2026", "2026-03-15", "proxy")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))

	srcFile := filepath.Join(srcDir, "GH010042.lrv")
	require.NoError(t, os.WriteFile(srcFile, []byte("gopro proxy"), 0o644))

	err := proxy.ProcessProxy(srcFile, proxyDir)

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(proxyDir, "GH010042.mp4"))
	assert.NoFileExists(t, srcFile)
}

func TestProcessProxy_OriginalFilenamePreserved(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "temp")
	proxyDir := filepath.Join(tmpDir, "media", "2026", "2026-03-15", "proxy")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))

	srcFile := filepath.Join(srcDir, "MY_CUSTOM_NAME.lrf")
	require.NoError(t, os.WriteFile(srcFile, []byte("custom proxy"), 0o644))

	err := proxy.ProcessProxy(srcFile, proxyDir)

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(proxyDir, "MY_CUSTOM_NAME.mp4"))
	assert.NoFileExists(t, srcFile)
}
