package watcher

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/pinn/takesort/internal/safepath"
)

// Watcher monitors a directory for new files using fsnotify.
type Watcher struct {
	events     chan string
	watchDir   string
	ignoreDirs []string
	logger     *slog.Logger
}

// New creates a Watcher that monitors watchDir and ignores events in ignoreDirs.
func New(watchDir string, ignoreDirs []string, logger *slog.Logger) *Watcher {
	return &Watcher{
		events:     make(chan string, 64),
		watchDir:   watchDir,
		ignoreDirs: ignoreDirs,
		logger:     logger,
	}
}

// Start begins watching for file creation events. It blocks until ctx is cancelled.
func (w *Watcher) Start(ctx context.Context) error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer func() { _ = fsw.Close() }()

	if err := fsw.Add(w.watchDir); err != nil {
		return err
	}

	// Walk existing subdirectories and add them to the watcher
	_ = filepath.WalkDir(w.watchDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() || path == w.watchDir {
			return nil
		}
		if isHidden(d.Name()) {
			w.logger.Info("deleting hidden directory", "path", path)
			if removeErr := os.RemoveAll(path); removeErr != nil {
				w.logger.Error("failed to delete hidden directory", "path", path, "error", removeErr)
			}
			return filepath.SkipDir
		}
		if w.isIgnored(path) {
			return filepath.SkipDir
		}
		if addErr := fsw.Add(path); addErr != nil {
			w.logger.Error("failed to watch subdirectory", "path", path, "error", addErr)
		}
		return nil
	})

	w.logger.Info("watcher started", "dir", w.watchDir)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("watcher stopping")
			return nil
		case event, ok := <-fsw.Events:
			if !ok {
				return nil
			}
			if !event.Has(fsnotify.Create) {
				continue
			}
			if w.isIgnored(event.Name) {
				w.logger.Debug("ignoring event in excluded dir", "path", event.Name)
				continue
			}

			// Check if the created path is a directory or file
			info, lstatErr := os.Lstat(event.Name)
			if lstatErr != nil {
				w.logger.Debug("lstat failed for event path", "path", event.Name, "error", lstatErr)
				continue
			}

			if info.IsDir() {
				w.handleNewDirectory(fsw, event.Name)
				continue
			}

			// Skip symlinks
			if info.Mode()&os.ModeSymlink != 0 {
				w.logger.Warn("ignoring symlink", "path", event.Name)
				continue
			}

			w.logger.Info("file detected", "path", event.Name)
			w.events <- event.Name

		case err, ok := <-fsw.Errors:
			if !ok {
				return nil
			}
			w.logger.Error("watcher error", "error", err)
		}
	}
}

// handleNewDirectory processes a newly created directory: hidden dirs are deleted,
// non-ignored dirs are added to the watcher, and any existing files inside are emitted.
func (w *Watcher) handleNewDirectory(fsw *fsnotify.Watcher, dirPath string) {
	name := filepath.Base(dirPath)

	if isHidden(name) {
		w.logger.Info("deleting hidden directory", "path", dirPath)
		if err := os.RemoveAll(dirPath); err != nil {
			w.logger.Error("failed to delete hidden directory", "path", dirPath, "error", err)
		}
		return
	}

	// Walk the new directory to watch all subdirs and emit existing files
	_ = filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			if isHidden(d.Name()) {
				w.logger.Info("deleting hidden directory", "path", path)
				if removeErr := os.RemoveAll(path); removeErr != nil {
					w.logger.Error("failed to delete hidden directory", "path", path, "error", removeErr)
				}
				return filepath.SkipDir
			}
			if w.isIgnored(path) {
				return filepath.SkipDir
			}
			if addErr := fsw.Add(path); addErr != nil {
				w.logger.Error("failed to watch subdirectory", "path", path, "error", addErr)
			}
			return nil
		}

		// Skip symlinks
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}

		if safepath.IsSymlink(path) {
			w.logger.Warn("ignoring symlink", "path", path)
			return nil
		}

		w.logger.Info("file detected", "path", path)
		w.events <- path
		return nil
	})
}

// Events returns a read-only channel that emits paths of newly created files.
func (w *Watcher) Events() <-chan string {
	return w.events
}

func (w *Watcher) isIgnored(path string) bool {
	cleanPath := filepath.Clean(path)
	for _, dir := range w.ignoreDirs {
		cleanDir := filepath.Clean(dir)
		if strings.HasPrefix(cleanPath, cleanDir+string(filepath.Separator)) || cleanPath == cleanDir {
			return true
		}
	}
	return false
}

func isHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}
