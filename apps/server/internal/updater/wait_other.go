//go:build !windows

package updater

import (
	"fmt"
	"time"
)

func waitForParentExit(pid int, _ time.Duration) error {
	if pid <= 0 {
		return nil
	}
	return fmt.Errorf("waiting for a Windows parent process is unsupported on this platform")
}
