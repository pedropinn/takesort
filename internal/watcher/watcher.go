package watcher

import (
	"context"
	"log/slog"
	"strings"

	"github.com/fsnotify/fsnotify"
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
	defer fsw.Close()

	if err := fsw.Add(w.watchDir); err != nil {
		return err
	}

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

// Events returns a read-only channel that emits paths of newly created files.
func (w *Watcher) Events() <-chan string {
	return w.events
}

func (w *Watcher) isIgnored(path string) bool {
	for _, dir := range w.ignoreDirs {
		if strings.HasPrefix(path, dir) {
			return true
		}
	}
	return false
}
