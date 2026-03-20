package errsink

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pinn/takesort/internal/mover"
)

// MoveToErrors moves src into errorsDir, creating it if needed.
func MoveToErrors(src string, errorsDir string) error {
	if err := os.MkdirAll(errorsDir, 0o755); err != nil {
		return fmt.Errorf("create errors dir: %w", err)
	}

	if err := mover.MoveFile(src, errorsDir); err != nil {
		return fmt.Errorf("move to errors: %w", err)
	}

	return nil
}

// ShouldRejectFile checks whether a file should be rejected before processing.
// Returns (true, reason) if the file is invalid, or (false, "") if valid.
func ShouldRejectFile(path string) (bool, string) {
	info, err := os.Stat(path)
	if err != nil {
		return true, fmt.Sprintf("cannot stat file: %v", err)
	}

	if info.Size() == 0 {
		return true, "zero size file"
	}

	if filepath.Ext(path) == "" {
		return true, "file without extension"
	}

	return false, ""
}
