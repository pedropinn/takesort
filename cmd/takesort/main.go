package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/pinn/takesort/internal/classifier"
	"github.com/pinn/takesort/internal/cleanup"
	"github.com/pinn/takesort/internal/config"
	"github.com/pinn/takesort/internal/conflict"
	"github.com/pinn/takesort/internal/errsink"
	"github.com/pinn/takesort/internal/logger"
	"github.com/pinn/takesort/internal/mover"
	"github.com/pinn/takesort/internal/orchestrator"
	"github.com/pinn/takesort/internal/proxy"
	"github.com/pinn/takesort/internal/trash"
	"github.com/pinn/takesort/internal/watcher"
)

var version = "dev"

// Adapters bridge package-level functions to the orchestrator interfaces.

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

type unknownDeleterAdapter struct{}

func (unknownDeleterAdapter) DeleteUnknown(path string) error {
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

type dirCleanerAdapter struct{}

func (dirCleanerAdapter) CleanEmptyDirs(rootDir string, ignoreDirs []string) error {
	return cleanup.CleanEmptyDirs(rootDir, ignoreDirs)
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	log := logger.Setup(cfg.LogLevel)

	log.Info("takesort starting",
		"version", version,
		"debounce_interval", cfg.DebounceInterval,
		"debounce_checks", cfg.DebounceChecks,
		"log_level", cfg.LogLevel,
		"watch_dir", cfg.WatchDir,
		"media_dir", cfg.MediaDir,
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	w := watcher.New(
		cfg.WatchDir,
		[]string{cfg.ConflictsDir, cfg.ErrorsDir},
		log,
	)

	orch := orchestrator.New(orchestrator.Deps{
		Classifier:       classifierAdapter{},
		Validator:        validatorAdapter{},
		Mover:            moverAdapter{},
		Proxy:            proxyAdapter{},
		Trash:            trashAdapter{},
		UnknownDeleter:   unknownDeleterAdapter{},
		Conflict:         conflictAdapter{},
		ErrorSink:        errsinkAdapter{},
		Watcher:          w,
		DirCleaner:       dirCleanerAdapter{},
		Logger:           log,
		MediaDir:         cfg.MediaDir,
		WatchDir:         cfg.WatchDir,
		ConflictDir:      cfg.ConflictsDir,
		ErrorsDir:        cfg.ErrorsDir,
		DebounceInterval: cfg.DebounceInterval,
		DebounceChecks:   cfg.DebounceChecks,
	})

	// Start watcher in a separate goroutine with proper shutdown tracking
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := w.Start(ctx); err != nil {
			log.Error("watcher error", "error", err)
		}
	}()

	if err := orch.Run(ctx); err != nil {
		log.Error("orchestrator error", "error", err)
		os.Exit(1)
	}

	wg.Wait()
	log.Info("takesort shutdown complete")
}
