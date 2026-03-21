package safepath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrSymlink is returned when a symlink is detected where only regular files are allowed.
var ErrSymlink = fmt.Errorf("symlinks are not allowed")

// IsSymlink reports whether path is a symbolic link.
func IsSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

// ValidateFile checks that path is a regular file (not a symlink) and resides
// within the allowedBase directory. Returns an error if validation fails.
func ValidateFile(path, allowedBase string) error {
	if IsSymlink(path) {
		return fmt.Errorf("%w: %s", ErrSymlink, path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve absolute path: %w", err)
	}

	absBase, err := filepath.Abs(allowedBase)
	if err != nil {
		return fmt.Errorf("resolve base path: %w", err)
	}

	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) && absPath != absBase {
		return fmt.Errorf("path %s is outside allowed directory %s", absPath, absBase)
	}

	return nil
}
