package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pinn/takesort/internal/classifier"
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

const version = "0.1.0"

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
		"log_level", cfg.LogLevel,
		"watch_dir", config.WatchDir,
		"media_dir", config.MediaDir,
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	w := watcher.New(
		config.WatchDir,
		[]string{config.ConflictsDir, config.ErrorsDir},
		log,
	)

	orch := orchestrator.New(orchestrator.Deps{
		Classifier:       classifierAdapter{},
		Validator:        validatorAdapter{},
		Mover:            moverAdapter{},
		Proxy:            proxyAdapter{},
		Trash:            trashAdapter{},
		Conflict:         conflictAdapter{},
		ErrorSink:        errsinkAdapter{},
		Watcher:          w,
		Logger:           log,
		MediaDir:         config.MediaDir,
		WatchDir:         config.WatchDir,
		ConflictDir:      config.ConflictsDir,
		ErrorsDir:        config.ErrorsDir,
		DebounceInterval: cfg.DebounceInterval,
		DebounceChecks:   4,
	})

	// Start watcher in a separate goroutine
	go func() {
		if err := w.Start(ctx); err != nil {
			log.Error("watcher error", "error", err)
		}
	}()

	if err := orch.Run(ctx); err != nil {
		log.Error("orchestrator error", "error", err)
		os.Exit(1)
	}

	log.Info("takesort shutdown complete")
}
