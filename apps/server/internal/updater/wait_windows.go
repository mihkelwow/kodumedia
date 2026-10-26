//go:build windows

package updater

import (
	"fmt"
	"syscall"
	"time"
)

const (
	processSynchronize = 0x00100000
	waitObject0        = 0x00000000
	waitTimeout        = 0x00000102
)

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	openProcess         = kernel32.NewProc("OpenProcess")
	waitForSingleObject = kernel32.NewProc("WaitForSingleObject")
	closeHandle         = kernel32.NewProc("CloseHandle")
)

func waitForParentExit(pid int, timeout time.Duration) error {
	if pid <= 0 {
		return nil
	}
	handle, _, callErr := openProcess.Call(processSynchronize, 0, uintptr(uint32(pid)))
	if handle == 0 {
		// The process may already have exited between plan creation and helper startup.
		if errno, ok := callErr.(syscall.Errno); ok && errno == syscall.Errno(87) {
			return nil
		}
		return fmt.Errorf("OpenProcess: %w", callErr)
	}
	defer closeHandle.Call(handle)
	milliseconds := uint32(timeout / time.Millisecond)
	result, _, callErr := waitForSingleObject.Call(handle, uintptr(milliseconds))
	switch result {
	case waitObject0:
		return nil
	case waitTimeout:
		return fmt.Errorf("timeout after %s", timeout)
	default:
		return fmt.Errorf("WaitForSingleObject result %d: %w", result, callErr)
	}
}
