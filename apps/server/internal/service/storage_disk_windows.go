//go:build windows

package service

import (
	"syscall"
	"unsafe"
)

var storageGetDiskFreeSpaceExW = syscall.NewLazyDLL("kernel32.dll").NewProc("GetDiskFreeSpaceExW")

func storageDiskSpace(path string) (freeBytes, totalBytes uint64, err error) {
	pointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, err
	}
	var available, total, free uint64
	result, _, callErr := storageGetDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(pointer)),
		uintptr(unsafe.Pointer(&available)),
		uintptr(unsafe.Pointer(&total)),
		uintptr(unsafe.Pointer(&free)),
	)
	if result == 0 {
		return 0, 0, callErr
	}
	return available, total, nil
}
