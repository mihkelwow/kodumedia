//go:build windows

package nativepicker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

const (
	coinitApartmentThreaded = 0x2
	clsctxInprocServer      = 0x1

	fosNoChangeDir     = 0x00000008
	fosPickFolders     = 0x00000020
	fosForceFileSystem = 0x00000040
	fosPathMustExist   = 0x00000800
	fosFileMustExist   = 0x00001000

	sigdnFileSystemPath = 0x80058000
	hresultCancelled    = 0x800704c7
	rpcChangedMode      = 0x80010106
)

var (
	clsidFileOpenDialog = syscall.GUID{Data1: 0xdc1c5a9c, Data2: 0xe88a, Data3: 0x4dde, Data4: [8]byte{0xa5, 0xa1, 0x60, 0xf8, 0x2a, 0x20, 0xae, 0xf7}}
	iidFileOpenDialog   = syscall.GUID{Data1: 0xd57c7288, Data2: 0xd4ad, Data3: 0x4768, Data4: [8]byte{0xbe, 0x02, 0x9d, 0x96, 0x95, 0x32, 0xd9, 0x60}}
	iidShellItem        = syscall.GUID{Data1: 0x43826d1e, Data2: 0xe718, Data3: 0x42ee, Data4: [8]byte{0xbc, 0x55, 0xa1, 0xe2, 0x61, 0xc3, 0x7b, 0xfe}}

	ole32                   = syscall.NewLazyDLL("ole32.dll")
	coInitializeEx          = ole32.NewProc("CoInitializeEx")
	coUninitialize          = ole32.NewProc("CoUninitialize")
	coCreateInstance        = ole32.NewProc("CoCreateInstance")
	coTaskMemFree           = ole32.NewProc("CoTaskMemFree")
	shell32                 = syscall.NewLazyDLL("shell32.dll")
	shCreateItemFromParsing = shell32.NewProc("SHCreateItemFromParsingName")
	user32                  = syscall.NewLazyDLL("user32.dll")
	getForegroundWindow     = user32.NewProc("GetForegroundWindow")
)

type fileOpenDialog struct {
	vtable *fileOpenDialogVTable
}

type fileOpenDialogVTable struct {
	queryInterface      uintptr
	addRef              uintptr
	release             uintptr
	show                uintptr
	setFileTypes        uintptr
	setFileTypeIndex    uintptr
	getFileTypeIndex    uintptr
	advise              uintptr
	unadvise            uintptr
	setOptions          uintptr
	getOptions          uintptr
	setDefaultFolder    uintptr
	setFolder           uintptr
	getFolder           uintptr
	getCurrentSelection uintptr
	setFileName         uintptr
	getFileName         uintptr
	setTitle            uintptr
	setOKButtonLabel    uintptr
	setFileNameLabel    uintptr
	getResult           uintptr
	addPlace            uintptr
	setDefaultExtension uintptr
	close               uintptr
	setClientGUID       uintptr
	clearClientData     uintptr
	setFilter           uintptr
	getResults          uintptr
	getSelectedItems    uintptr
}

type shellItem struct {
	vtable *shellItemVTable
}

type shellItemVTable struct {
	queryInterface uintptr
	addRef         uintptr
	release        uintptr
	bindToHandler  uintptr
	getParent      uintptr
	getDisplayName uintptr
	getAttributes  uintptr
	compare        uintptr
}

type Picker struct {
	mu sync.Mutex
}

func New() *Picker { return &Picker{} }

func (p *Picker) Pick(ctx context.Context, request domain.FilePickerRequest) (string, bool, error) {
	if err := validateRequest(request); err != nil {
		return "", false, err
	}
	select {
	case <-ctx.Done():
		return "", false, ctx.Err()
	default:
	}

	// Windows shell dialogs require a single-threaded COM apartment. Locking the
	// goroutine keeps all COM calls on the same native thread.
	p.mu.Lock()
	defer p.mu.Unlock()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hr, _, _ := coInitializeEx.Call(0, coinitApartmentThreaded)
	initialized := !hresultFailed(hr)
	if hresultFailed(hr) && uint32(hr) != rpcChangedMode {
		return "", false, hresultError("CoInitializeEx", hr)
	}
	if initialized {
		defer coUninitialize.Call()
	}

	var dialog *fileOpenDialog
	hr, _, _ = coCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidFileOpenDialog)),
		0,
		clsctxInprocServer,
		uintptr(unsafe.Pointer(&iidFileOpenDialog)),
		uintptr(unsafe.Pointer(&dialog)),
	)
	if hresultFailed(hr) || dialog == nil {
		return "", false, hresultError("CoCreateInstance(IFileOpenDialog)", hr)
	}
	defer dialog.release()

	options := uintptr(fosNoChangeDir | fosForceFileSystem | fosPathMustExist)
	if request.Mode == domain.FilePickerModeDirectory {
		options |= fosPickFolders
	} else {
		options |= fosFileMustExist
	}
	if hr = dialog.call(dialog.vtable.setOptions, options); hresultFailed(hr) {
		return "", false, hresultError("IFileDialog.SetOptions", hr)
	}
	if title := strings.TrimSpace(request.Title); title != "" {
		titlePointer, err := syscall.UTF16PtrFromString(title)
		if err != nil {
			return "", false, err
		}
		if hr = dialog.call(dialog.vtable.setTitle, uintptr(unsafe.Pointer(titlePointer))); hresultFailed(hr) {
			return "", false, hresultError("IFileDialog.SetTitle", hr)
		}
	}
	setInitialFolder(dialog, request.InitialPath)

	owner, _, _ := getForegroundWindow.Call()
	hr = dialog.call(dialog.vtable.show, owner)
	if uint32(hr) == hresultCancelled {
		return "", true, nil
	}
	if hresultFailed(hr) {
		return "", false, hresultError("IFileDialog.Show", hr)
	}

	var item *shellItem
	hr = dialog.call(dialog.vtable.getResult, uintptr(unsafe.Pointer(&item)))
	if hresultFailed(hr) || item == nil {
		return "", false, hresultError("IFileDialog.GetResult", hr)
	}
	defer item.release()

	var pathPointer *uint16
	hr, _, _ = syscall.SyscallN(
		item.vtable.getDisplayName,
		uintptr(unsafe.Pointer(item)),
		sigdnFileSystemPath,
		uintptr(unsafe.Pointer(&pathPointer)),
	)
	if hresultFailed(hr) || pathPointer == nil {
		return "", false, hresultError("IShellItem.GetDisplayName", hr)
	}
	defer coTaskMemFree.Call(uintptr(unsafe.Pointer(pathPointer)))
	path := utf16String(pathPointer)
	if path == "" {
		return "", false, fmt.Errorf("%w: Windows returned an empty path", domain.ErrUnavailable)
	}
	if request.Mode == domain.FilePickerModeFile && !extensionAllowed(path, request.Extensions) {
		return "", false, fmt.Errorf("%w: valitud failit\u00fc\u00fcp ei ole lubatud", domain.ErrInvalidInput)
	}
	return filepath.Clean(path), false, nil
}

func validateRequest(request domain.FilePickerRequest) error {
	if request.Mode != domain.FilePickerModeDirectory && request.Mode != domain.FilePickerModeFile {
		return fmt.Errorf("%w: tundmatu failivalija re\u017eiim", domain.ErrInvalidInput)
	}
	return nil
}

func setInitialFolder(dialog *fileOpenDialog, initial string) {
	initial = strings.TrimSpace(initial)
	if initial == "" {
		return
	}
	if info, err := os.Stat(initial); err == nil && !info.IsDir() {
		initial = filepath.Dir(initial)
	}
	if info, err := os.Stat(initial); err != nil || !info.IsDir() {
		return
	}
	pathPointer, err := syscall.UTF16PtrFromString(initial)
	if err != nil {
		return
	}
	var item *shellItem
	hr, _, _ := shCreateItemFromParsing.Call(
		uintptr(unsafe.Pointer(pathPointer)),
		0,
		uintptr(unsafe.Pointer(&iidShellItem)),
		uintptr(unsafe.Pointer(&item)),
	)
	if hresultFailed(hr) || item == nil {
		return
	}
	defer item.release()
	_ = dialog.call(dialog.vtable.setFolder, uintptr(unsafe.Pointer(item)))
}

func extensionAllowed(path string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	extension := strings.ToLower(filepath.Ext(path))
	for _, value := range allowed {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if !strings.HasPrefix(value, ".") {
			value = "." + value
		}
		if extension == value {
			return true
		}
	}
	return false
}

func (d *fileOpenDialog) call(method uintptr, arguments ...uintptr) uintptr {
	values := make([]uintptr, 1, len(arguments)+1)
	values[0] = uintptr(unsafe.Pointer(d))
	values = append(values, arguments...)
	result, _, _ := syscall.SyscallN(method, values...)
	return result
}

func (d *fileOpenDialog) release() {
	if d != nil && d.vtable != nil {
		_, _, _ = syscall.SyscallN(d.vtable.release, uintptr(unsafe.Pointer(d)))
	}
}

func (item *shellItem) release() {
	if item != nil && item.vtable != nil {
		_, _, _ = syscall.SyscallN(item.vtable.release, uintptr(unsafe.Pointer(item)))
	}
}

func utf16String(pointer *uint16) string {
	if pointer == nil {
		return ""
	}
	values := make([]uint16, 0, 260)
	base := unsafe.Pointer(pointer)
	for index := uintptr(0); index < 32768; index++ {
		value := *(*uint16)(unsafe.Add(base, index*2))
		if value == 0 {
			break
		}
		values = append(values, value)
	}
	return syscall.UTF16ToString(values)
}

func hresultFailed(value uintptr) bool { return int32(uint32(value)) < 0 }

func hresultError(operation string, value uintptr) error {
	return fmt.Errorf("%s failed with HRESULT 0x%08X", operation, uint32(value))
}

var _ ports.FilePicker = (*Picker)(nil)
