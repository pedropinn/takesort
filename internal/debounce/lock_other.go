//go:build !unix

package debounce

// isFileLocked is a no-op on non-Unix platforms.
func isFileLocked(_ string) bool {
	return false
}
