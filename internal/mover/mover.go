package mover

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/pinn/takesort/internal/classifier"
)

// subfolders maps each FileType to its destination subdirectory.
var subfolders = map[classifier.FileType]string{
	classifier.Video:     "",
	classifier.Proxy:     "proxy",
	classifier.PhotoJPEG: filepath.Join("photos", "jpeg"),
	classifier.PhotoRAW:  filepath.Join("photos", "raw"),
	classifier.Audio:     "audio",
}

// BuildDestPath returns the destination directory for a file based on its
// modification time and type. The layout is {mediaDir}/{YYYY}/{YYYY-MM-DD}/<subfolder>.
func BuildDestPath(mediaDir string, modTime time.Time, fileType classifier.FileType) string {
	year := modTime.Format("2006")
	date := modTime.Format("2006-01-02")

	base := filepath.Join(mediaDir, year, date)

	sub, ok := subfolders[fileType]
	if !ok || sub == "" {
		return base
	}
	return filepath.Join(base, sub)
}

// MoveFile moves the file at src into destDir, creating destDir if needed.
// Falls back to copy+delete when src and destDir are on different devices.
func MoveFile(src string, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}

	dst := filepath.Join(destDir, filepath.Base(src))

	if err := os.Rename(src, dst); err != nil {
		if isCrossDeviceError(err) {
			return copyAndDelete(src, dst)
		}
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// isCrossDeviceError reports whether err is caused by an EXDEV cross-device link.
func isCrossDeviceError(err error) bool {
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		if errno, ok := linkErr.Err.(syscall.Errno); ok {
			return errno == syscall.EXDEV
		}
	}
	return false
}

// copyAndDelete copies src to dst preserving permissions, then removes src.
func copyAndDelete(src, dst string) error {
	if err := copyFile(src, dst); err != nil {
		return err
	}
	if err := os.Remove(src); err != nil {
		return fmt.Errorf("remove source after copy: %w", err)
	}
	return nil
}

// copyFile copies the content and permissions of src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer srcFile.Close()

	info, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return fmt.Errorf("create dest: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy content: %w", err)
	}

	if err := dstFile.Close(); err != nil {
		return fmt.Errorf("close dest: %w", err)
	}
	return nil
}
