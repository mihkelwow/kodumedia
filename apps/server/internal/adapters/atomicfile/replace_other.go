//go:build !windows

package atomicfile

import "os"

// Replace atomically moves source over destination on platforms where rename
// replaces an existing file.
func Replace(source, destination string) error {
	return os.Rename(source, destination)
}
