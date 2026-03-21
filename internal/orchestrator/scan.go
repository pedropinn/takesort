package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
)

// ScanExisting lists regular files in the root of dir, excluding files
// whose paths fall inside any of the ignoreDirs directories.
func ScanExisting(dir string, ignoreDirs []string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() || e.Type()&os.ModeSymlink != 0 {
			continue
		}

		abs := filepath.Join(dir, e.Name())
		if isInsideIgnored(abs, ignoreDirs) {
			continue
		}

		files = append(files, abs)
	}

	return files, nil
}

func isInsideIgnored(path string, ignoreDirs []string) bool {
	for _, dir := range ignoreDirs {
		if strings.HasPrefix(path, dir+string(filepath.Separator)) || path == dir {
			return true
		}
	}
	return false
}
