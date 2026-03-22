package orchestrator

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// ScanExisting recursively walks dir, returning all regular files found at any
// depth. Hidden directories (name starts with '.') are deleted with os.RemoveAll
// and skipped. Directories matching or inside ignoreDirs are skipped. Symlinks
// are excluded.
func ScanExisting(dir string, ignoreDirs []string, logger *slog.Logger) ([]string, error) {
	var files []string

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			// Never process the root directory itself
			if path == dir {
				return nil
			}

			// Hidden directories: delete entirely and skip
			if strings.HasPrefix(d.Name(), ".") {
				logger.Info("deleting hidden directory", "path", path)
				if removeErr := os.RemoveAll(path); removeErr != nil {
					logger.Error("failed to delete hidden directory", "path", path, "error", removeErr)
				}
				return filepath.SkipDir
			}

			// Ignored directories: skip without deleting
			if isInsideIgnored(path, ignoreDirs) {
				return filepath.SkipDir
			}

			return nil
		}

		// Skip symlinks
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}

		// Skip files inside ignored directories
		if isInsideIgnored(path, ignoreDirs) {
			return nil
		}

		files = append(files, path)
		return nil
	})

	return files, err
}

func isInsideIgnored(path string, ignoreDirs []string) bool {
	for _, dir := range ignoreDirs {
		if strings.HasPrefix(path, dir+string(filepath.Separator)) || path == dir {
			return true
		}
	}
	return false
}
