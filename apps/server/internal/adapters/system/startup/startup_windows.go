//go:build windows

package startup

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

const (
	hkeyCurrentUser   = uintptr(0x80000001)
	keyQueryValue     = 0x0001
	keySetValue       = 0x0002
	regSZ             = 1
	regBinary         = 3
	errorSuccess      = 0
	errorFileNotFound = 2
	errorPathNotFound = 3
)

var (
	advapi32            = syscall.NewLazyDLL("advapi32.dll")
	procRegOpenKeyEx    = advapi32.NewProc("RegOpenKeyExW")
	procRegCreateKeyEx  = advapi32.NewProc("RegCreateKeyExW")
	procRegQueryValueEx = advapi32.NewProc("RegQueryValueExW")
	procRegSetValueEx   = advapi32.NewProc("RegSetValueExW")
	procRegDeleteValue  = advapi32.NewProc("RegDeleteValueW")
	procRegCloseKey     = advapi32.NewProc("RegCloseKey")
)

func New() (*Manager, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable: %w", err)
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return nil, fmt.Errorf("resolve executable path: %w", err)
	}
	return newManager(filepath.Clean(executable), windowsRegistry{}), nil
}

type windowsRegistry struct{}

func (windowsRegistry) ReadString(path, name string) (string, error) {
	valueType, data, err := readRegistryValue(path, name)
	if err != nil {
		return "", err
	}
	if valueType != regSZ || len(data) < 2 || len(data)%2 != 0 {
		return "", fmt.Errorf("invalid registry string value")
	}
	buffer := make([]uint16, len(data)/2)
	for index := range buffer {
		buffer[index] = uint16(data[index*2]) | uint16(data[index*2+1])<<8
	}
	return syscall.UTF16ToString(buffer), nil
}

func (windowsRegistry) ReadBinary(path, name string) ([]byte, error) {
	valueType, data, err := readRegistryValue(path, name)
	if err != nil {
		return nil, err
	}
	if valueType != regBinary || len(data) == 0 {
		return nil, fmt.Errorf("invalid registry binary value")
	}
	return data, nil
}

func (windowsRegistry) WriteString(path, name, value string) error {
	key, err := openRegistryKey(path, true, keyQueryValue|keySetValue)
	if err != nil {
		return err
	}
	defer procRegCloseKey.Call(key)
	namePointer, _ := syscall.UTF16PtrFromString(name)
	encoded, _ := syscall.UTF16FromString(value)
	result, _, _ := procRegSetValueEx.Call(
		key,
		uintptr(unsafe.Pointer(namePointer)),
		0,
		regSZ,
		uintptr(unsafe.Pointer(&encoded[0])),
		uintptr(len(encoded)*2),
	)
	if result != errorSuccess {
		return registryError(result)
	}
	return nil
}

func (windowsRegistry) DeleteValue(path, name string) error {
	key, err := openRegistryKey(path, false, keySetValue)
	if err != nil {
		return err
	}
	defer procRegCloseKey.Call(key)
	namePointer, _ := syscall.UTF16PtrFromString(name)
	result, _, _ := procRegDeleteValue.Call(key, uintptr(unsafe.Pointer(namePointer)))
	if result != errorSuccess {
		return registryError(result)
	}
	return nil
}

func readRegistryValue(path, name string) (uint32, []byte, error) {
	key, err := openRegistryKey(path, false, keyQueryValue)
	if err != nil {
		return 0, nil, err
	}
	defer procRegCloseKey.Call(key)
	namePointer, _ := syscall.UTF16PtrFromString(name)
	var valueType uint32
	var size uint32
	result, _, _ := procRegQueryValueEx.Call(key, uintptr(unsafe.Pointer(namePointer)), 0, uintptr(unsafe.Pointer(&valueType)), 0, uintptr(unsafe.Pointer(&size)))
	if result != errorSuccess {
		return 0, nil, registryError(result)
	}
	if size == 0 || size > 32*1024 {
		return 0, nil, fmt.Errorf("invalid registry value size")
	}
	buffer := make([]byte, size)
	result, _, _ = procRegQueryValueEx.Call(
		key,
		uintptr(unsafe.Pointer(namePointer)),
		0,
		uintptr(unsafe.Pointer(&valueType)),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if result != errorSuccess {
		return 0, nil, registryError(result)
	}
	return valueType, buffer[:size], nil
}

func openRegistryKey(path string, create bool, access uint32) (uintptr, error) {
	pathPointer, _ := syscall.UTF16PtrFromString(path)
	var key uintptr
	if create {
		var disposition uint32
		result, _, _ := procRegCreateKeyEx.Call(
			hkeyCurrentUser,
			uintptr(unsafe.Pointer(pathPointer)),
			0,
			0,
			0,
			uintptr(access),
			0,
			uintptr(unsafe.Pointer(&key)),
			uintptr(unsafe.Pointer(&disposition)),
		)
		if result != errorSuccess {
			return 0, registryError(result)
		}
		return key, nil
	}
	result, _, _ := procRegOpenKeyEx.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(pathPointer)), 0, uintptr(access), uintptr(unsafe.Pointer(&key)))
	if result != errorSuccess {
		return 0, registryError(result)
	}
	return key, nil
}

func registryError(result uintptr) error {
	if result == errorFileNotFound || result == errorPathNotFound {
		return errRegistryValueNotFound
	}
	return syscall.Errno(result)
}
