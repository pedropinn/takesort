package orchestrator_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pinn/takesort/internal/classifier"
	"github.com/pinn/takesort/internal/conflict"
	"github.com/pinn/takesort/internal/errsink"
	"github.com/pinn/takesort/internal/mover"
	"github.com/pinn/takesort/internal/orchestrator"
	"github.com/pinn/takesort/internal/proxy"
	"github.com/pinn/takesort/internal/trash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Adapter types to satisfy the orchestrator interfaces using the existing packages.

type classifierAdapter struct{}

func (classifierAdapter) Classify(filename string) classifier.FileType {
	return classifier.Classify(filename)
}

type validatorAdapter struct{}

func (validatorAdapter) ShouldReject(path string) (bool, string) {
	return errsink.ShouldRejectFile(path)
}

type moverAdapter struct{}

func (moverAdapter) MoveFile(src, destDir string) error {
	return mover.MoveFile(src, destDir)
}

func (moverAdapter) BuildDestPath(mediaDir string, modTime time.Time, ft classifier.FileType) string {
	return mover.BuildDestPath(mediaDir, modTime, ft)
}

type proxyAdapter struct{}

func (proxyAdapter) ProcessProxy(src, destDir string) error {
	return proxy.ProcessProxy(src, destDir)
}

type trashAdapter struct{}

func (trashAdapter) DeleteTrash(path string) error {
	return trash.DeleteTrash(path)
}

type conflictAdapter struct{}

func (conflictAdapter) HasConflict(destPath string) bool {
	return conflict.HasConflict(destPath)
}

func (conflictAdapter) MoveToConflicts(src, conflictsDir string) error {
	return conflict.MoveToConflicts(src, conflictsDir)
}

type errsinkAdapter struct{}

func (errsinkAdapter) MoveToErrors(src, errorsDir string) error {
	return errsink.MoveToErrors(src, errorsDir)
}

// fakeEvents implements orchestrator.EventSource with a channel.
type fakeEvents struct {
	ch chan string
}

func (f *fakeEvents) Events() <-chan string {
	return f.ch
}

func newTestOrchestrator(mediaDir, watchDir, conflictDir, errorsDir string) *orchestrator.Orchestrator {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	return orchestrator.New(orchestrator.Deps{
		Classifier:       classifierAdapter{},
		Validator:        validatorAdapter{},
		Mover:            moverAdapter{},
		Proxy:            proxyAdapter{},
		Trash:            trashAdapter{},
		Conflict:         conflictAdapter{},
		ErrorSink:        errsinkAdapter{},
		Watcher:          &fakeEvents{ch: make(chan string)},
		Logger:           logger,
		MediaDir:         mediaDir,
		WatchDir:         watchDir,
		ConflictDir:      conflictDir,
		ErrorsDir:        errorsDir,
		DebounceInterval: 10 * time.Millisecond,
		DebounceChecks:   1,
	})
}

func TestProcessFile_MP4ClassifiedAndMovedToDateFolder(t *testing.T) {
	tmpDir := t.TempDir()
	watchDir := filepath.Join(tmpDir, "temp")
	mediaDir := filepath.Join(tmpDir, "media")
	conflictDir := filepath.Join(tmpDir, "temp", "conflicts")
	errorsDir := filepath.Join(tmpDir, "temp", "errors")
	require.NoError(t, os.MkdirAll(watchDir, 0o755))

	srcFile := filepath.Join(watchDir, "clip.mp4")
	require.NoError(t, os.WriteFile(srcFile, []byte("video content"), 0o644))

	// Set a known mod time
	modTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(srcFile, modTime, modTime))

	orch := newTestOrchestrator(mediaDir, watchDir, conflictDir, errorsDir)
	err := orch.ProcessFile(srcFile)

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(mediaDir, "2026", "2026-03-15", "clip.mp4"))
	assert.NoFileExists(t, srcFile)
}

func TestProcessFile_LRFRenamedAndMovedToProxy(t *testing.T) {
	tmpDir := t.TempDir()
	watchDir := filepath.Join(tmpDir, "temp")
	mediaDir := filepath.Join(tmpDir, "media")
	conflictDir := filepath.Join(tmpDir, "temp", "conflicts")
	errorsDir := filepath.Join(tmpDir, "temp", "errors")
	require.NoError(t, os.MkdirAll(watchDir, 0o755))

	srcFile := filepath.Join(watchDir, "DJI_0001.lrf")
	require.NoError(t, os.WriteFile(srcFile, []byte("proxy data"), 0o644))

	modTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(srcFile, modTime, modTime))

	orch := newTestOrchestrator(mediaDir, watchDir, conflictDir, errorsDir)
	err := orch.ProcessFile(srcFile)

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(mediaDir, "2026", "2026-03-15", "proxy", "DJI_0001.mp4"))
	assert.NoFileExists(t, srcFile)
}

func TestProcessFile_THMDeletedAfterClassification(t *testing.T) {
	tmpDir := t.TempDir()
	watchDir := filepath.Join(tmpDir, "temp")
	mediaDir := filepath.Join(tmpDir, "media")
	conflictDir := filepath.Join(tmpDir, "temp", "conflicts")
	errorsDir := filepath.Join(tmpDir, "temp", "errors")
	require.NoError(t, os.MkdirAll(watchDir, 0o755))
	require.NoError(t, os.MkdirAll(mediaDir, 0o755))

	srcFile := filepath.Join(watchDir, "DJI_0001.thm")
	require.NoError(t, os.WriteFile(srcFile, []byte("thumbnail"), 0o644))

	orch := newTestOrchestrator(mediaDir, watchDir, conflictDir, errorsDir)
	err := orch.ProcessFile(srcFile)

	require.NoError(t, err)
	assert.NoFileExists(t, srcFile)
}

func TestProcessFile_UnknownExtensionMovedToErrors(t *testing.T) {
	tmpDir := t.TempDir()
	watchDir := filepath.Join(tmpDir, "temp")
	mediaDir := filepath.Join(tmpDir, "media")
	conflictDir := filepath.Join(tmpDir, "temp", "conflicts")
	errorsDir := filepath.Join(tmpDir, "temp", "errors")
	require.NoError(t, os.MkdirAll(watchDir, 0o755))
	require.NoError(t, os.MkdirAll(mediaDir, 0o755))

	srcFile := filepath.Join(watchDir, "readme.xyz")
	require.NoError(t, os.WriteFile(srcFile, []byte("unknown"), 0o644))

	orch := newTestOrchestrator(mediaDir, watchDir, conflictDir, errorsDir)
	err := orch.ProcessFile(srcFile)

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(errorsDir, "readme.xyz"))
	assert.NoFileExists(t, srcFile)
}
