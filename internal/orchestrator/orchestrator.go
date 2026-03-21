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

// Deps holds all injected dependencies for the Orchestrator.
type Deps struct {
	Classifier       FileClassifier
	Validator        FileValidator
	Mover            FileMover
	Proxy            ProxyProcessor
	Trash            TrashDeleter
	Conflict         ConflictChecker
	ErrorSink        ErrorSink
	Watcher          EventSource
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
		logger.Warn("symlink rejected")
		return o.deps.ErrorSink.MoveToErrors(path, o.deps.ErrorsDir)
	}

	// 1. Validate
	if reject, reason := o.deps.Validator.ShouldReject(path); reject {
		logger.Warn("file rejected", "reason", reason)
		return o.deps.ErrorSink.MoveToErrors(path, o.deps.ErrorsDir)
	}

	// 2. Classify
	ft := o.deps.Classifier.Classify(path)
	logger = logger.With("type", ft)
	logger.Info("file classified")

	// 3. Unknown -> errors
	if ft == classifier.Unknown {
		logger.Warn("unknown file type, moving to errors")
		return o.deps.ErrorSink.MoveToErrors(path, o.deps.ErrorsDir)
	}

	// 4. Trash -> delete
	if ft == classifier.Trash {
		logger.Info("deleting trash file")
		return o.deps.Trash.DeleteTrash(path)
	}

	// 5. Get ModTime (Lstat to avoid following symlinks)
	info, err := os.Lstat(path)
	if err != nil {
		logger.Error("stat failed", "error", err)
		return o.deps.ErrorSink.MoveToErrors(path, o.deps.ErrorsDir)
	}
	modTime := info.ModTime()

	// 6. Build dest path
	destDir := o.deps.Mover.BuildDestPath(o.deps.MediaDir, modTime, ft)

	// 7. Check conflict
	destFile := filepath.Join(destDir, filepath.Base(path))
	if ft == classifier.Proxy {
		ext := filepath.Ext(path)
		baseName := filepath.Base(path)
		nameNoExt := baseName[:len(baseName)-len(ext)]
		destFile = filepath.Join(destDir, nameNoExt+".mp4")
	}

	if o.deps.Conflict.HasConflict(destFile) {
		logger.Warn("conflict detected, moving to conflicts", "dest", destFile)
		return o.deps.Conflict.MoveToConflicts(path, o.deps.ConflictDir)
	}

	// 8-9. Move proxy
	if ft == classifier.Proxy {
		logger.Info("processing proxy", "dest", destDir)
		if err := o.deps.Proxy.ProcessProxy(path, destDir); err != nil {
			logger.Error("proxy processing failed", "error", err)
			return o.deps.ErrorSink.MoveToErrors(path, o.deps.ErrorsDir)
		}
		return nil
	}

	// 10. Regular move
	logger.Info("moving file", "dest", destDir)
	if err := o.deps.Mover.MoveFile(path, destDir); err != nil {
		logger.Error("move failed", "error", err)
		return o.deps.ErrorSink.MoveToErrors(path, o.deps.ErrorsDir)
	}

	return nil
}

// Run starts the orchestrator loop, consuming events until ctx is cancelled.
// It first scans for orphan files left over from previous runs, then enters
// the watcher event loop.
func (o *Orchestrator) Run(ctx context.Context) error {
	// Create required directories
	for _, dir := range []string{o.deps.ConflictDir, o.deps.ErrorsDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}

	// Validate media dir access
	if _, err := os.Stat(o.deps.MediaDir); err != nil {
		o.deps.Logger.Error("media dir inaccessible", "dir", o.deps.MediaDir, "error", err)
		return fmt.Errorf("FATAL: media dir inaccessible: %w", err)
	}

	o.deps.Logger.Info("orchestrator started",
		"media", o.deps.MediaDir,
		"watch", o.deps.WatchDir,
	)

	// Process orphan files left in watch dir from previous runs
	orphans, err := ScanExisting(o.deps.WatchDir, []string{o.deps.ConflictDir, o.deps.ErrorsDir})
	if err != nil {
		o.deps.Logger.Error("scan existing files failed", "error", err)
	} else if len(orphans) > 0 {
		o.deps.Logger.Info("processing orphan files", "count", len(orphans))
		for _, path := range orphans {
			if err := debounce.WaitForStability(path, o.deps.DebounceInterval, o.deps.DebounceChecks); err != nil {
				if err == debounce.ErrFileDisappeared {
					o.deps.Logger.Warn("orphan file disappeared during debounce", "path", path)
					continue
				}
				o.deps.Logger.Error("orphan debounce error", "path", path, "error", err)
				continue
			}
			if err := o.ProcessFile(path); err != nil {
				o.deps.Logger.Error("orphan process error", "path", path, "error", err)
			}
		}
	}

	events := o.deps.Watcher.Events()
	for {
		select {
		case <-ctx.Done():
			o.deps.Logger.Info("orchestrator stopping")
			return nil
		case path, ok := <-events:
			if !ok {
				return nil
			}
			if err := debounce.WaitForStability(path, o.deps.DebounceInterval, o.deps.DebounceChecks); err != nil {
				if err == debounce.ErrFileDisappeared {
					o.deps.Logger.Warn("file disappeared during debounce", "path", path)
					continue
				}
				o.deps.Logger.Error("debounce error", "path", path, "error", err)
				continue
			}
			if err := o.ProcessFile(path); err != nil {
				o.deps.Logger.Error("process file error", "path", path, "error", err)
			}
		}
	}
}
