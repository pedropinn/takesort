package conflict

import (
	"fmt"
	"os"

	"github.com/pinn/takesort/internal/mover"
)

// HasConflict reports whether a file already exists at destPath.
func HasConflict(destPath string) bool {
	_, err := os.Stat(destPath)
	return err == nil
}

// MoveToConflicts moves src into conflictsDir, creating it if needed.
func MoveToConflicts(src string, conflictsDir string) error {
	if err := os.MkdirAll(conflictsDir, 0o755); err != nil {
		return fmt.Errorf("create conflicts dir: %w", err)
	}

	if err := mover.MoveFile(src, conflictsDir); err != nil {
		return fmt.Errorf("move to conflicts: %w", err)
	}

	return nil
}
