package orchestrator_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pinn/takesort/internal/classifier"
	"github.com/pinn/takesort/internal/mover"
	"github.com/pinn/takesort/internal/orchestrator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupDirs creates the standard temp/media/conflicts/errors directory layout
// for integration tests and returns the paths.
func setupDirs(t *testing.T) (watchDir, mediaDir, conflictDir, errorsDir string) {
	t.Helper()
	root := t.TempDir()
	watchDir = filepath.Join(root, "temp")
	mediaDir = filepath.Join(root, "media")
	conflictDir = filepath.Join(root, "temp", "conflicts")
	errorsDir = filepath.Join(root, "temp", "errors")
	require.NoError(t, os.MkdirAll(watchDir, 0o755))
	require.NoError(t, os.MkdirAll(mediaDir, 0o755))
	require.NoError(t, os.MkdirAll(conflictDir, 0o755))
	require.NoError(t, os.MkdirAll(errorsDir, 0o755))
	return
}

func newIntegrationOrchestrator(watchDir, mediaDir, conflictDir, errorsDir string) (*orchestrator.Orchestrator, *fakeEvents) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	fe := &fakeEvents{ch: make(chan string, 16)}
	orch := orchestrator.New(orchestrator.Deps{
		Classifier:       classifierAdapter{},
		Validator:        validatorAdapter{},
		Mover:            moverAdapter{},
		Proxy:            proxyAdapter{},
		Trash:            trashAdapter{},
		Conflict:         conflictAdapter{},
		ErrorSink:        errsinkAdapter{},
		Watcher:          fe,
		Logger:           logger,
		MediaDir:         mediaDir,
		WatchDir:         watchDir,
		ConflictDir:      conflictDir,
		ErrorsDir:        errorsDir,
		DebounceInterval: 10 * time.Millisecond,
		DebounceChecks:   1,
	})
	return orch, fe
}

func TestIntegration_FullMP4Flow(t *testing.T) {
	watchDir, mediaDir, conflictDir, errorsDir := setupDirs(t)
	orch, fe := newIntegrationOrchestrator(watchDir, mediaDir, conflictDir, errorsDir)

	srcFile := filepath.Join(watchDir, "DJI_0042.mp4")
	require.NoError(t, os.WriteFile(srcFile, []byte("video content"), 0o644))

	modTime := time.Date(2026, 6, 10, 14, 30, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(srcFile, modTime, modTime))

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		fe.ch <- srcFile
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	err := orch.Run(ctx)
	require.NoError(t, err)

	destFile := filepath.Join(mediaDir, "2026", "2026-06-10", "DJI_0042.mp4")
	assert.FileExists(t, destFile)
	assert.NoFileExists(t, srcFile)
}

func TestIntegration_ProxyLRFFlow(t *testing.T) {
	watchDir, mediaDir, conflictDir, errorsDir := setupDirs(t)
	orch, fe := newIntegrationOrchestrator(watchDir, mediaDir, conflictDir, errorsDir)

	srcFile := filepath.Join(watchDir, "DJI_0001.lrf")
	require.NoError(t, os.WriteFile(srcFile, []byte("proxy data"), 0o644))

	modTime := time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(srcFile, modTime, modTime))

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		fe.ch <- srcFile
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	err := orch.Run(ctx)
	require.NoError(t, err)

	destFile := filepath.Join(mediaDir, "2026", "2026-04-20", "proxy", "DJI_0001.mp4")
	assert.FileExists(t, destFile)
	assert.NoFileExists(t, srcFile)
}

func TestIntegration_ConflictFlow(t *testing.T) {
	watchDir, mediaDir, conflictDir, errorsDir := setupDirs(t)
	orch, fe := newIntegrationOrchestrator(watchDir, mediaDir, conflictDir, errorsDir)

	modTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)

	// Place an existing file at the destination to create a conflict
	destDir := filepath.Join(mediaDir, "2026", "2026-03-15")
	require.NoError(t, os.MkdirAll(destDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(destDir, "clip.mp4"), []byte("existing"), 0o644))

	srcFile := filepath.Join(watchDir, "clip.mp4")
	require.NoError(t, os.WriteFile(srcFile, []byte("duplicate"), 0o644))
	require.NoError(t, os.Chtimes(srcFile, modTime, modTime))

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		fe.ch <- srcFile
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	err := orch.Run(ctx)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(conflictDir, "clip.mp4"))
	assert.NoFileExists(t, srcFile)
}

func TestIntegration_TrashTHMFlow(t *testing.T) {
	watchDir, mediaDir, conflictDir, errorsDir := setupDirs(t)
	orch, fe := newIntegrationOrchestrator(watchDir, mediaDir, conflictDir, errorsDir)

	srcFile := filepath.Join(watchDir, "DJI_0001.thm")
	require.NoError(t, os.WriteFile(srcFile, []byte("thumbnail"), 0o644))

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		fe.ch <- srcFile
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	err := orch.Run(ctx)
	require.NoError(t, err)

	assert.NoFileExists(t, srcFile)
}

func TestIntegration_ZeroSizeFileToErrors(t *testing.T) {
	watchDir, mediaDir, conflictDir, errorsDir := setupDirs(t)
	orch, fe := newIntegrationOrchestrator(watchDir, mediaDir, conflictDir, errorsDir)

	srcFile := filepath.Join(watchDir, "empty.mp4")
	require.NoError(t, os.WriteFile(srcFile, []byte{}, 0o644))

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		fe.ch <- srcFile
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	err := orch.Run(ctx)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(errorsDir, "empty.mp4"))
	assert.NoFileExists(t, srcFile)
}

func TestIntegration_NoExtensionFileToErrors(t *testing.T) {
	watchDir, mediaDir, conflictDir, errorsDir := setupDirs(t)
	orch, fe := newIntegrationOrchestrator(watchDir, mediaDir, conflictDir, errorsDir)

	srcFile := filepath.Join(watchDir, "noext")
	require.NoError(t, os.WriteFile(srcFile, []byte("some content"), 0o644))

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		fe.ch <- srcFile
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	err := orch.Run(ctx)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(errorsDir, "noext"))
	assert.NoFileExists(t, srcFile)
}

func TestIntegration_MultipleDatesOrganizedSeparately(t *testing.T) {
	watchDir, mediaDir, conflictDir, errorsDir := setupDirs(t)
	orch, fe := newIntegrationOrchestrator(watchDir, mediaDir, conflictDir, errorsDir)

	fileA := filepath.Join(watchDir, "clipA.mp4")
	fileB := filepath.Join(watchDir, "clipB.mp4")
	require.NoError(t, os.WriteFile(fileA, []byte("video A"), 0o644))
	require.NoError(t, os.WriteFile(fileB, []byte("video B"), 0o644))

	dateA := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	dateB := time.Date(2026, 7, 25, 16, 0, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(fileA, dateA, dateA))
	require.NoError(t, os.Chtimes(fileB, dateB, dateB))

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		fe.ch <- fileA
		fe.ch <- fileB
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	err := orch.Run(ctx)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(mediaDir, "2026", "2026-01-10", "clipA.mp4"))
	assert.FileExists(t, filepath.Join(mediaDir, "2026", "2026-07-25", "clipB.mp4"))
}

func TestIntegration_OrphanProcessingOnStartup(t *testing.T) {
	watchDir, mediaDir, conflictDir, errorsDir := setupDirs(t)

	// Pre-populate watch dir with an orphan file
	orphanFile := filepath.Join(watchDir, "orphan.mp4")
	require.NoError(t, os.WriteFile(orphanFile, []byte("orphan video"), 0o644))

	modTime := time.Date(2026, 2, 14, 8, 0, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(orphanFile, modTime, modTime))

	orch, _ := newIntegrationOrchestrator(watchDir, mediaDir, conflictDir, errorsDir)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(300 * time.Millisecond)
		cancel()
	}()

	err := orch.Run(ctx)
	require.NoError(t, err)

	destFile := filepath.Join(mediaDir, "2026", "2026-02-14", "orphan.mp4")
	assert.FileExists(t, destFile)
	assert.NoFileExists(t, orphanFile)
}

func TestIntegration_ErrorFlowMoveFailsGoesToErrors(t *testing.T) {
	watchDir, mediaDir, conflictDir, errorsDir := setupDirs(t)

	srcFile := filepath.Join(watchDir, "video.mp4")
	require.NoError(t, os.WriteFile(srcFile, []byte("content"), 0o644))

	modTime := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(srcFile, modTime, modTime))

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	fe := &fakeEvents{ch: make(chan string, 16)}

	orch := orchestrator.New(orchestrator.Deps{
		Classifier:       classifierAdapter{},
		Validator:        validatorAdapter{},
		Mover:            failingMoverAdapter{},
		Proxy:            proxyAdapter{},
		Trash:            trashAdapter{},
		Conflict:         conflictAdapter{},
		ErrorSink:        errsinkAdapter{},
		Watcher:          fe,
		Logger:           logger,
		MediaDir:         mediaDir,
		WatchDir:         watchDir,
		ConflictDir:      conflictDir,
		ErrorsDir:        errorsDir,
		DebounceInterval: 10 * time.Millisecond,
		DebounceChecks:   1,
	})

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		fe.ch <- srcFile
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	err := orch.Run(ctx)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(errorsDir, "video.mp4"))
	assert.NoFileExists(t, srcFile)
}

func TestIntegration_CrossDeviceMoveFallback(t *testing.T) {
	watchDir, mediaDir, conflictDir, errorsDir := setupDirs(t)
	orch, fe := newIntegrationOrchestrator(watchDir, mediaDir, conflictDir, errorsDir)

	srcFile := filepath.Join(watchDir, "footage.mp4")
	content := []byte("video footage for cross-device test")
	require.NoError(t, os.WriteFile(srcFile, content, 0o644))

	modTime := time.Date(2026, 8, 5, 11, 0, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(srcFile, modTime, modTime))

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		fe.ch <- srcFile
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	err := orch.Run(ctx)
	require.NoError(t, err)

	destFile := filepath.Join(mediaDir, "2026", "2026-08-05", "footage.mp4")
	assert.FileExists(t, destFile)
	assert.NoFileExists(t, srcFile)

	// Verify content integrity after move
	data, err := os.ReadFile(destFile)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

// failingMoverAdapter always fails on MoveFile but delegates BuildDestPath.
type failingMoverAdapter struct{}

func (failingMoverAdapter) MoveFile(_, _ string) error {
	return os.ErrPermission
}

func (failingMoverAdapter) BuildDestPath(mediaDir string, modTime time.Time, ft classifier.FileType) string {
	return mover.BuildDestPath(mediaDir, modTime, ft)
}
