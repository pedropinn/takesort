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
	errorsDir = filepath.Join(root, "temp", "conflicts", "errors")
	require.NoError(t, os.MkdirAll(watchDir, 0o755))
	require.NoError(t, os.MkdirAll(mediaDir, 0o755))
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
		UnknownDeleter:   unknownDeleterAdapter{},
		Conflict:         conflictAdapter{},
		ErrorSink:        errsinkAdapter{},
		Watcher:          fe,
		DirCleaner:       dirCleanerAdapter{},
		DateExtractor:    dateExtractAdapter{},
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
		UnknownDeleter:   unknownDeleterAdapter{},
		Conflict:         conflictAdapter{},
		ErrorSink:        errsinkAdapter{},
		Watcher:          fe,
		DirCleaner:       dirCleanerAdapter{},
		DateExtractor:    dateExtractAdapter{},
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

// Recursive SD card ingestion integration tests

func TestIntegration_RecursiveOrphansProcessedAndDirsCleanedOnStartup(t *testing.T) {
	watchDir, mediaDir, conflictDir, errorsDir := setupDirs(t)

	// Simulate an SD card structure already present at startup
	dcimDir := filepath.Join(watchDir, "DCIM", "100MEDIA")
	require.NoError(t, os.MkdirAll(dcimDir, 0o755))

	orphanFile := filepath.Join(dcimDir, "DJI_0001.mp4")
	require.NoError(t, os.WriteFile(orphanFile, []byte("drone video"), 0o644))

	modTime := time.Date(2026, 5, 20, 14, 0, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(orphanFile, modTime, modTime))

	orch, _ := newIntegrationOrchestrator(watchDir, mediaDir, conflictDir, errorsDir)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(300 * time.Millisecond)
		cancel()
	}()

	err := orch.Run(ctx)
	require.NoError(t, err)

	// File should be moved to date-based destination
	assert.FileExists(t, filepath.Join(mediaDir, "2026", "2026-05-20", "DJI_0001.mp4"))
	assert.NoFileExists(t, orphanFile)

	// Empty directory tree should be cleaned up
	assert.NoDirExists(t, dcimDir)
	assert.NoDirExists(t, filepath.Join(watchDir, "DCIM"))
}

func TestIntegration_SDCardStructureMixedTypesAndHiddenDirs(t *testing.T) {
	watchDir, mediaDir, conflictDir, errorsDir := setupDirs(t)

	modTime := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)

	// Create a realistic SD card structure
	dcimDir := filepath.Join(watchDir, "DCIM", "100MEDIA")
	require.NoError(t, os.MkdirAll(dcimDir, 0o755))

	// Hidden directories (macOS artifacts)
	hiddenTrashDir := filepath.Join(watchDir, ".Trashes")
	require.NoError(t, os.MkdirAll(hiddenTrashDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(hiddenTrashDir, "junk.dat"), []byte("junk"), 0o644))

	hiddenSpotlight := filepath.Join(watchDir, ".Spotlight-V100")
	require.NoError(t, os.MkdirAll(hiddenSpotlight, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(hiddenSpotlight, "index.db"), []byte("index"), 0o644))

	// Media files at various locations
	mp4File := filepath.Join(dcimDir, "DJI_0042.mp4")
	require.NoError(t, os.WriteFile(mp4File, []byte("video"), 0o644))
	require.NoError(t, os.Chtimes(mp4File, modTime, modTime))

	jpgFile := filepath.Join(dcimDir, "DJI_0042.jpg")
	require.NoError(t, os.WriteFile(jpgFile, []byte("photo"), 0o644))
	require.NoError(t, os.Chtimes(jpgFile, modTime, modTime))

	// Trash file
	thmFile := filepath.Join(dcimDir, "DJI_0042.thm")
	require.NoError(t, os.WriteFile(thmFile, []byte("thumbnail"), 0o644))

	// Unknown file (SD card junk)
	logFile := filepath.Join(dcimDir, "MISC.log")
	require.NoError(t, os.WriteFile(logFile, []byte("log data"), 0o644))

	orch, _ := newIntegrationOrchestrator(watchDir, mediaDir, conflictDir, errorsDir)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(400 * time.Millisecond)
		cancel()
	}()

	err := orch.Run(ctx)
	require.NoError(t, err)

	// Media files sorted to correct destinations
	assert.FileExists(t, filepath.Join(mediaDir, "2026", "2026-09-01", "DJI_0042.mp4"))
	assert.FileExists(t, filepath.Join(mediaDir, "2026", "2026-09-01", "photos", "jpeg", "DJI_0042.jpg"))

	// Trash file deleted
	assert.NoFileExists(t, thmFile)

	// Unknown file deleted (not moved to errors)
	assert.NoFileExists(t, logFile)
	assert.NoFileExists(t, filepath.Join(errorsDir, "MISC.log"))

	// Hidden directories deleted
	assert.NoDirExists(t, hiddenTrashDir)
	assert.NoDirExists(t, hiddenSpotlight)

	// Empty directory tree cleaned up
	assert.NoDirExists(t, dcimDir)
	assert.NoDirExists(t, filepath.Join(watchDir, "DCIM"))
}

func TestIntegration_HiddenDirInsideSubdirectoryDeletedDuringScan(t *testing.T) {
	watchDir, mediaDir, conflictDir, errorsDir := setupDirs(t)

	// Create a hidden directory nested inside a subdirectory
	dcimDir := filepath.Join(watchDir, "DCIM")
	hiddenInDCIM := filepath.Join(dcimDir, ".fseventsd")
	require.NoError(t, os.MkdirAll(hiddenInDCIM, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(hiddenInDCIM, "fseventsd-uuid"), []byte("uuid"), 0o644))

	// Also add a media file so we exercise the full pipeline
	mediaSubdir := filepath.Join(dcimDir, "100MEDIA")
	require.NoError(t, os.MkdirAll(mediaSubdir, 0o755))
	mp4File := filepath.Join(mediaSubdir, "GH010042.mp4")
	require.NoError(t, os.WriteFile(mp4File, []byte("gopro video"), 0o644))
	modTime := time.Date(2026, 11, 15, 8, 0, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(mp4File, modTime, modTime))

	orch, _ := newIntegrationOrchestrator(watchDir, mediaDir, conflictDir, errorsDir)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(300 * time.Millisecond)
		cancel()
	}()

	err := orch.Run(ctx)
	require.NoError(t, err)

	// Hidden directory inside DCIM should be deleted
	assert.NoDirExists(t, hiddenInDCIM)

	// Media file processed
	assert.FileExists(t, filepath.Join(mediaDir, "2026", "2026-11-15", "GH010042.mp4"))

	// Empty tree cleaned up
	assert.NoDirExists(t, dcimDir)
}

func TestIntegration_DeepDirectoryTreeCleanedAfterProcessing(t *testing.T) {
	watchDir, mediaDir, conflictDir, errorsDir := setupDirs(t)

	// Create a deep nested structure: DCIM/100MEDIA/sub1/sub2/
	deepDir := filepath.Join(watchDir, "DCIM", "100MEDIA", "sub1", "sub2")
	require.NoError(t, os.MkdirAll(deepDir, 0o755))

	// Put a single file at the deepest level
	wavFile := filepath.Join(deepDir, "recording.wav")
	require.NoError(t, os.WriteFile(wavFile, []byte("audio data"), 0o644))
	modTime := time.Date(2026, 7, 4, 16, 0, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(wavFile, modTime, modTime))

	orch, _ := newIntegrationOrchestrator(watchDir, mediaDir, conflictDir, errorsDir)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(300 * time.Millisecond)
		cancel()
	}()

	err := orch.Run(ctx)
	require.NoError(t, err)

	// File processed
	assert.FileExists(t, filepath.Join(mediaDir, "2026", "2026-07-04", "audio", "recording.wav"))

	// Entire deep directory tree cleaned up
	assert.NoDirExists(t, deepDir)
	assert.NoDirExists(t, filepath.Join(watchDir, "DCIM"))
}

// DJI filename date extraction integration tests

func TestIntegration_DJIFilenameDateRoutedViaWatcherEvent(t *testing.T) {
	watchDir, mediaDir, conflictDir, errorsDir := setupDirs(t)
	orch, fe := newIntegrationOrchestrator(watchDir, mediaDir, conflictDir, errorsDir)

	srcFile := filepath.Join(watchDir, "DJI_20260218094732_0004_D.MP4")
	require.NoError(t, os.WriteFile(srcFile, []byte("drone video"), 0o644))

	// Set ModTime to a different date to verify filename date wins
	modTime := time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(srcFile, modTime, modTime))

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		fe.ch <- srcFile
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	err := orch.Run(ctx)
	require.NoError(t, err)

	// File should be routed by filename date (2026-02-18), not ModTime (2026-03-22)
	assert.FileExists(t, filepath.Join(mediaDir, "2026", "2026-02-18", "DJI_20260218094732_0004_D.MP4"))
	assert.NoFileExists(t, srcFile)
	assert.NoFileExists(t, filepath.Join(mediaDir, "2026", "2026-03-22", "DJI_20260218094732_0004_D.MP4"))
}

func TestIntegration_NonDJIFileFallbackToModTimeViaWatcherEvent(t *testing.T) {
	watchDir, mediaDir, conflictDir, errorsDir := setupDirs(t)
	orch, fe := newIntegrationOrchestrator(watchDir, mediaDir, conflictDir, errorsDir)

	srcFile := filepath.Join(watchDir, "GoPro_0042.mp4")
	require.NoError(t, os.WriteFile(srcFile, []byte("gopro video"), 0o644))

	modTime := time.Date(2026, 5, 10, 14, 0, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(srcFile, modTime, modTime))

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		fe.ch <- srcFile
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	err := orch.Run(ctx)
	require.NoError(t, err)

	// Non-DJI file should fall back to ModTime routing
	assert.FileExists(t, filepath.Join(mediaDir, "2026", "2026-05-10", "GoPro_0042.mp4"))
	assert.NoFileExists(t, srcFile)
}

func TestIntegration_DJINamedJPGRoutedByFilenameDateViaWatcherEvent(t *testing.T) {
	watchDir, mediaDir, conflictDir, errorsDir := setupDirs(t)
	orch, fe := newIntegrationOrchestrator(watchDir, mediaDir, conflictDir, errorsDir)

	srcFile := filepath.Join(watchDir, "DJI_20260218094732_0004_D.JPG")
	require.NoError(t, os.WriteFile(srcFile, []byte("drone photo"), 0o644))

	// Set ModTime to a different date to verify filename date wins
	modTime := time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(srcFile, modTime, modTime))

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		fe.ch <- srcFile
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	err := orch.Run(ctx)
	require.NoError(t, err)

	// JPG should be routed to photos/jpeg using filename date
	assert.FileExists(t, filepath.Join(mediaDir, "2026", "2026-02-18", "photos", "jpeg", "DJI_20260218094732_0004_D.JPG"))
	assert.NoFileExists(t, srcFile)
}

func TestIntegration_DJINamedOrphanRoutedByFilenameDateOnStartup(t *testing.T) {
	watchDir, mediaDir, conflictDir, errorsDir := setupDirs(t)

	// Pre-populate watch dir with a DJI-named orphan file
	orphanFile := filepath.Join(watchDir, "DJI_20260218094732_0004_D.MP4")
	require.NoError(t, os.WriteFile(orphanFile, []byte("drone video"), 0o644))

	// Set ModTime to a different date to verify filename date wins
	modTime := time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(orphanFile, modTime, modTime))

	orch, _ := newIntegrationOrchestrator(watchDir, mediaDir, conflictDir, errorsDir)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(300 * time.Millisecond)
		cancel()
	}()

	err := orch.Run(ctx)
	require.NoError(t, err)

	// Orphan file should be routed by filename date (2026-02-18), not ModTime (2026-03-22)
	assert.FileExists(t, filepath.Join(mediaDir, "2026", "2026-02-18", "DJI_20260218094732_0004_D.MP4"))
	assert.NoFileExists(t, orphanFile)
}
