package proxy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pinn/takesort/internal/mover"
)

// ProcessProxy renames a proxy file (.lrf/.lrv) to .mp4 and moves it to destDir.
func ProcessProxy(src string, destDir string) error {
	base := filepath.Base(src)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	renamedFile := filepath.Join(filepath.Dir(src), name+".mp4")

	if err := os.Rename(src, renamedFile); err != nil {
		return fmt.Errorf("rename proxy extension: %w", err)
	}

	if err := mover.MoveFile(renamedFile, destDir); err != nil {
		return fmt.Errorf("move proxy: %w", err)
	}

	return nil
}
