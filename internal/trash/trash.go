package trash

import (
	"fmt"
	"os"
)

// DeleteTrash removes the file at path from the filesystem.
func DeleteTrash(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete trash: %w", err)
	}
	return nil
}
