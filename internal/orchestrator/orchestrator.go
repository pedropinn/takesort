package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/pinn/takesort/internal/classifier"
	"github.com/pinn/takesort/internal/debounce"
	"github.com/pinn/takesort/internal/logmsg"
	"github.com/pinn/takesort/internal/safepath"
)

// FileClassifier classifies a file by its name.
type FileClassifier interface {
	Classify(filename string) classifier.FileType
}

// FileValidator checks whether a file should be rejected.
type FileValidator interface {
	ShouldReject(path string) (bool, string)
}

// FileMover moves a file to a destination directory.
type FileMover interface {
	MoveFile(src, destDir string) error
	BuildDestPath(mediaDir string, modTime time.Time, fileType classifier.FileType) string
}

// ProxyProcessor renames and moves a proxy file.
type ProxyProcessor interface {
	ProcessProxy(src, destDir string) error
}

// TrashDeleter deletes a trash file.
type TrashDeleter interface {
	DeleteTrash(path string) error
}

// UnknownDeleter deletes an unknown file.
type UnknownDeleter interface {
	DeleteUnknown(path string) error
}

// ConflictChecker detects and handles file name conflicts.
type ConflictChecker interface {
	HasConflict(destPath string) bool
	MoveToConflicts(src, conflictsDir string) error
}

// ErrorSink moves problematic files to the errors directory.
type ErrorSink interface {
	MoveToErrors(src, errorsDir string) error
}

// EventSource provides a channel of file paths to process.
type EventSource interface {
	Events() <-chan string
}

// DirCleaner removes empty directories from a root directory.
type DirCleaner interface {
	CleanEmptyDirs(rootDir string, ignoreDirs []string) error
}

// DateExtractor extracts a date from a filename.
type DateExtractor interface {
	ExtractDate(filename string) (time.Time, bool)
}

// Deps holds all injected dependencies for the Orchestrator.
type Deps struct {
	Classifier       FileClassifier
	Validator        FileValidator
	Mover            FileMover
	Proxy            ProxyProcessor
	Trash            TrashDeleter
	UnknownDeleter   UnknownDeleter
	Conflict         ConflictChecker
	ErrorSink        ErrorSink
	Watcher          EventSource
	DirCleaner       DirCleaner
	DateExtractor    DateExtractor
	Logger           *slog.Logger
	MediaDir         string
	WatchDir         string
	ConflictDir      string
	ErrorsDir        string
	DebounceInterval time.Duration
	DebounceChecks   int
}

// Orchestrator coordinates the file processing pipeline.
type Orchestrator struct {
	deps Deps
}

// New creates an Orchestrator with the provided dependencies.
func New(deps Deps) *Orchestrator {
	if deps.DebounceChecks == 0 {
		deps.DebounceChecks = 4
	}
	return &Orchestrator{deps: deps}
}

// ProcessFile runs the full pipeline for a single file.
func (o *Orchestrator) ProcessFile(path string) error {
	logger := o.deps.Logger.With("file", filepath.Base(path))

	// 0. Reject symlinks
	if safepath.IsSymlink(path) {
		logger.Warn(logmsg.FileSymlinkRejected)
		return o.deps.ErrorSink.MoveToErrors(path, o.deps.ErrorsDir)
	}

	// 1. Validate
	if reject, reason := o.deps.Validator.ShouldReject(path); reject {
		logger.Warn(logmsg.FileValidationFailed, "reason", reason)
		return o.deps.ErrorSink.MoveToErrors(path, o.deps.ErrorsDir)
	}

	// 2. Classify
	ft := o.deps.Classifier.Classify(path)
	logger = logger.With("type", ft)
	logger.Info(logmsg.FileClassified)

	// 3. Unknown -> delete
	if ft == classifier.Unknown {
		logger.Warn(logmsg.FileUnknownDeleting)
		return o.deps.UnknownDeleter.DeleteUnknown(path)
	}

	// 4. Trash -> delete
	if ft == classifier.Trash {
		logger.Info(logmsg.FileTrashDeleting)
		return o.deps.Trash.DeleteTrash(path)
	}

	// 5. Resolve date: try filename extraction first, fall back to ModTime
	var fileDate time.Time
	var dateSource string
	if o.deps.DateExtractor != nil {
		if extracted, ok := o.deps.DateExtractor.ExtractDate(filepath.Base(path)); ok {
			fileDate = extracted
			dateSource = "filename"
		}
	}
	if fileDate.IsZero() {
		info, err := os.Lstat(path)
		if err != nil {
			logger.Error(logmsg.FileStatFailed, "error", err)
			return o.deps.ErrorSink.MoveToErrors(path, o.deps.ErrorsDir)
		}
		fileDate = info.ModTime()
		dateSource = "modtime"
	}
	logger.Info(logmsg.FileDateResolved, "date", fileDate.Format("2006-01-02"), "source", dateSource)

	// 6. Build dest path
	destDir := o.deps.Mover.BuildDestPath(o.deps.MediaDir, fileDate, ft)

	// 7. Check conflict
	destFile := filepath.Join(destDir, filepath.Base(path))
	if ft == classifier.Proxy {
		ext := filepath.Ext(path)
		baseName := filepath.Base(path)
		nameNoExt := baseName[:len(baseName)-len(ext)]
		destFile = filepath.Join(destDir, nameNoExt+".mp4")
	}

	if o.deps.Conflict.HasConflict(destFile) {
		logger.Warn(logmsg.FileConflictDetected, "dest", destFile)
		return o.deps.Conflict.MoveToConflicts(path, o.deps.ConflictDir)
	}

	// 8-9. Move proxy
	if ft == classifier.Proxy {
		logger.Info(logmsg.FileProxyProcessing, "dest", destDir, "date", fileDate.Format("2006-01-02"), "source", dateSource)
		if err := o.deps.Proxy.ProcessProxy(path, destDir); err != nil {
			logger.Error(logmsg.FileProxyFailed, "error", err)
			return o.deps.ErrorSink.MoveToErrors(path, o.deps.ErrorsDir)
		}
		return nil
	}

	// 10. Regular move
	logger.Info(logmsg.FileMoving, "dest", destDir, "date", fileDate.Format("2006-01-02"), "source", dateSource)
	if err := o.deps.Mover.MoveFile(path, destDir); err != nil {
		logger.Error(logmsg.FileMoveFailed, "error", err)
		return o.deps.ErrorSink.MoveToErrors(path, o.deps.ErrorsDir)
	}

	return nil
}

// Run starts the orchestrator loop, consuming events until ctx is cancelled.
// It first scans for orphan files left over from previous runs, then enters
// the watcher event loop.
func (o *Orchestrator) Run(ctx context.Context) error {
	// Validate media dir access
	if _, err := os.Stat(o.deps.MediaDir); err != nil {
		o.deps.Logger.Error(logmsg.OrchestratorMediaDirInaccessible, "dir", o.deps.MediaDir, "error", err)
		return fmt.Errorf("FATAL: media dir inaccessible: %w", err)
	}

	o.deps.Logger.Info(logmsg.OrchestratorStarted,
		"media", o.deps.MediaDir,
		"watch", o.deps.WatchDir,
	)

	ignoreDirs := []string{o.deps.ConflictDir}

	// Process orphan files left in watch dir from previous runs
	orphans, err := ScanExisting(o.deps.WatchDir, ignoreDirs, o.deps.Logger)
	if err != nil {
		o.deps.Logger.Error(logmsg.OrphanScanFailed, "error", err)
	} else if len(orphans) > 0 {
		o.deps.Logger.Info(logmsg.OrphanProcessingStarted, "count", len(orphans))
		for _, path := range orphans {
			o.deps.Logger.Debug(logmsg.DebounceStarted, "path", path)
			if err := debounce.WaitForStability(ctx, path, o.deps.DebounceInterval, o.deps.DebounceChecks); err != nil {
				if err == debounce.ErrFileDisappeared {
					o.deps.Logger.Warn(logmsg.OrphanDebounceDisappeared, "path", path)
					continue
				}
				o.deps.Logger.Error(logmsg.OrphanDebounceFailed, "path", path, "error", err)
				continue
			}
			o.deps.Logger.Debug(logmsg.DebounceComplete, "path", path)
			if err := o.ProcessFile(path); err != nil {
				o.deps.Logger.Error(logmsg.OrphanProcessFailed, "path", path, "error", err)
			}
		}
	}

	// Clean up any empty subdirectories left after orphan processing
	if err := o.deps.DirCleaner.CleanEmptyDirs(o.deps.WatchDir, ignoreDirs); err != nil {
		o.deps.Logger.Error(logmsg.CleanupAfterOrphansFailed, "error", err)
	}

	// Periodic cleanup of empty conflicts/errors directories
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := os.Remove(o.deps.ErrorsDir); err == nil {
					o.deps.Logger.Info(logmsg.CleanupErrorsDirRemoved)
				}
				if err := os.Remove(o.deps.ConflictDir); err == nil {
					o.deps.Logger.Info(logmsg.CleanupConflictsDirRemoved)
				}
			}
		}
	}()

	events := o.deps.Watcher.Events()
	for {
		select {
		case <-ctx.Done():
			o.deps.Logger.Info(logmsg.OrchestratorStopping)
			return nil
		case path, ok := <-events:
			if !ok {
				return nil
			}
			o.deps.Logger.Debug(logmsg.DebounceStarted, "path", path)
			if err := debounce.WaitForStability(ctx, path, o.deps.DebounceInterval, o.deps.DebounceChecks); err != nil {
				if err == debounce.ErrFileDisappeared {
					o.deps.Logger.Warn(logmsg.DebounceFileDisappeared, "path", path)
					continue
				}
				o.deps.Logger.Error(logmsg.DebounceFailed, "path", path, "error", err)
				continue
			}
			o.deps.Logger.Debug(logmsg.DebounceComplete, "path", path)
			if err := o.ProcessFile(path); err != nil {
				o.deps.Logger.Error(logmsg.FileProcessFailed, "path", path, "error", err)
			}

			// Clean up any empty subdirectories after processing
			if err := o.deps.DirCleaner.CleanEmptyDirs(o.deps.WatchDir, ignoreDirs); err != nil {
				o.deps.Logger.Error(logmsg.CleanupAfterFileFailed, "error", err)
			}
		}
	}
}
