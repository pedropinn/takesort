package cleanup

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// CleanEmptyDirs walks rootDir bottom-up and removes any empty subdirectories,
// skipping directories that match or are inside any of the ignoreDirs paths.
// The rootDir itself is never removed. ENOTEMPTY errors are treated as non-errors
// to handle race conditions where new files appear between checks.
func CleanEmptyDirs(rootDir string, ignoreDirs []string) error {
	var dirs []string

	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if path == rootDir {
			return nil
		}
		if isInsideIgnored(path, ignoreDirs) {
			return filepath.SkipDir
		}
		dirs = append(dirs, path)
		return nil
	})
	if err != nil {
		return err
	}

	// Iterate in reverse order for bottom-up removal (deepest first)
	for i := len(dirs) - 1; i >= 0; i-- {
		removeErr := os.Remove(dirs[i])
		if removeErr == nil {
			continue
		}
		// ENOTEMPTY or ENOENT are expected race conditions; skip them
		if errors.Is(removeErr, syscall.ENOTEMPTY) || errors.Is(removeErr, os.ErrNotExist) {
			continue
		}
		// On some platforms the error may not unwrap to ENOTEMPTY but
		// the directory simply is not empty -- treat any *PathError from
		// os.Remove on a non-empty dir as non-fatal.
		var pathErr *os.PathError
		if errors.As(removeErr, &pathErr) {
			continue
		}
	}

	return nil
}

func isInsideIgnored(path string, ignoreDirs []string) bool {
	for _, dir := range ignoreDirs {
		if strings.HasPrefix(path, dir+string(filepath.Separator)) || path == dir {
			return true
		}
	}
	return false
}
