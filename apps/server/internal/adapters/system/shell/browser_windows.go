//go:build windows

package shell

import (
	"fmt"
	"syscall"
	"unsafe"
)

const swShowNormal = 1

var (
	shell32          = syscall.NewLazyDLL("shell32.dll")
	procShellExecute = shell32.NewProc("ShellExecuteW")
)

func OpenURL(target string) error {
	verb, _ := syscall.UTF16PtrFromString("open")
	value, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	result, _, callErr := procShellExecute.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(value)), 0, 0, swShowNormal)
	if result <= 32 {
		return fmt.Errorf("open browser: result %d: %w", result, callErr)
	}
	return nil
}
