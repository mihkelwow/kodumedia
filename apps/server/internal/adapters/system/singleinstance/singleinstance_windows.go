//go:build windows

package singleinstance

import (
	"fmt"
	"syscall"
	"unsafe"
)

const errorAlreadyExists = syscall.Errno(183)

var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutex = kernel32.NewProc("CreateMutexW")
	procCloseHandle = kernel32.NewProc("CloseHandle")
)

type Lock struct {
	handle uintptr
}

func Acquire(name string) (*Lock, bool, error) {
	value, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return nil, false, err
	}
	handle, _, callErr := procCreateMutex.Call(0, 0, uintptr(unsafe.Pointer(value)))
	if handle == 0 {
		return nil, false, fmt.Errorf("create single-instance mutex: %w", callErr)
	}
	alreadyRunning := callErr == errorAlreadyExists
	if alreadyRunning {
		procCloseHandle.Call(handle)
		return nil, true, nil
	}
	return &Lock{handle: handle}, false, nil
}

func (l *Lock) Close() {
	if l == nil || l.handle == 0 {
		return
	}
	procCloseHandle.Call(l.handle)
	l.handle = 0
}
