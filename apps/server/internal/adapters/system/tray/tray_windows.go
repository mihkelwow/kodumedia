//go:build windows

package tray

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	wmDestroy     = 0x0002
	wmClose       = 0x0010
	wmNull        = 0x0000
	wmContextMenu = 0x007B
	wmLButtonUp   = 0x0202
	wmRButtonUp   = 0x0205
	wmApp         = 0x8000
	trayMessage   = wmApp + 1

	nimAdd    = 0x00000000
	nimModify = 0x00000001
	nimDelete = 0x00000002

	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004
	nifInfo    = 0x00000010

	niifInfo  = 0x00000001
	niifError = 0x00000003

	imageIcon      = 1
	lrDefaultSize  = 0x00000040
	lrLoadFromFile = 0x00000010

	mfString    = 0x00000000
	mfChecked   = 0x00000008
	mfSeparator = 0x00000800

	tpmRightButton = 0x0002
	tpmNonotify    = 0x0080
	tpmReturnCmd   = 0x0100

	swHide = 0

	menuOpenHome = 1001
	menuStartup  = 1002
	menuScanAll  = 1003
	menuSettings = 1004
	menuExit     = 1005

	initialIconAttempts = 31
	initialIconDelay    = time.Second
)

var (
	user32                    = syscall.NewLazyDLL("user32.dll")
	shell32                   = syscall.NewLazyDLL("shell32.dll")
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procRegisterClassEx       = user32.NewProc("RegisterClassExW")
	procRegisterWindowMessage = user32.NewProc("RegisterWindowMessageW")
	procUnregisterClass       = user32.NewProc("UnregisterClassW")
	procCreateWindowEx        = user32.NewProc("CreateWindowExW")
	procDestroyWindow         = user32.NewProc("DestroyWindow")
	procDefWindowProc         = user32.NewProc("DefWindowProcW")
	procGetMessage            = user32.NewProc("GetMessageW")
	procTranslateMsg          = user32.NewProc("TranslateMessage")
	procDispatchMsg           = user32.NewProc("DispatchMessageW")
	procPostMessage           = user32.NewProc("PostMessageW")
	procPostQuitMessage       = user32.NewProc("PostQuitMessage")
	procCreatePopupMenu       = user32.NewProc("CreatePopupMenu")
	procAppendMenu            = user32.NewProc("AppendMenuW")
	procDestroyMenu           = user32.NewProc("DestroyMenu")
	procSetMenuDefault        = user32.NewProc("SetMenuDefaultItem")
	procTrackPopupMenu        = user32.NewProc("TrackPopupMenu")
	procGetCursorPos          = user32.NewProc("GetCursorPos")
	procSetForeground         = user32.NewProc("SetForegroundWindow")
	procLoadImage             = user32.NewProc("LoadImageW")
	procDestroyIcon           = user32.NewProc("DestroyIcon")
	procShowWindow            = user32.NewProc("ShowWindow")
	procGetModuleHandle       = kernel32.NewProc("GetModuleHandleW")
	procShellNotifyIcon       = shell32.NewProc("Shell_NotifyIconW")
)

//go:embed assets/kodu-media.ico
var iconBytes []byte

type Callbacks struct {
	OpenHome       func()
	StartupEnabled func() bool
	ToggleStartup  func()
	ScanAll        func()
	OpenSettings   func()
	Exit           func()
}

type Tray struct {
	callbacks Callbacks
	dataDir   string

	mutex          sync.RWMutex
	hwnd           uintptr
	hicon          uintptr
	iconAdded      bool
	taskbarMessage uint32

	ready     chan struct{}
	done      chan struct{}
	readyOnce sync.Once
	doneOnce  sync.Once
}

type point struct {
	X int32
	Y int32
}

type message struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
	Private uint32
}

type wndClassEx struct {
	CbSize     uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSmall  uintptr
}

type notifyIconData struct {
	CbSize           uint32
	HWnd             uintptr
	ID               uint32
	Flags            uint32
	CallbackMessage  uint32
	Icon             uintptr
	Tip              [128]uint16
	State            uint32
	StateMask        uint32
	Info             [256]uint16
	TimeoutOrVersion uint32
	InfoTitle        [64]uint16
	InfoFlags        uint32
	GUIDItem         [16]byte
	BalloonIcon      uintptr
}

var (
	traysMu sync.RWMutex
	trays   = map[uintptr]*Tray{}
)

func New(dataDir string, callbacks Callbacks) *Tray {
	return &Tray{
		callbacks: callbacks,
		dataDir:   dataDir,
		ready:     make(chan struct{}),
		done:      make(chan struct{}),
	}
}

func (t *Tray) Run() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer t.signalDone()

	iconPath, err := t.ensureIconFile()
	if err != nil {
		t.signalReady()
		return err
	}
	instance, _, callErr := procGetModuleHandle.Call(0)
	if instance == 0 {
		t.signalReady()
		return fmt.Errorf("get module handle: %w", callErr)
	}
	className, _ := syscall.UTF16PtrFromString(fmt.Sprintf("KoduMediaTray_%d", os.Getpid()))
	windowName, _ := syscall.UTF16PtrFromString("KoduMedia")
	taskbarName, _ := syscall.UTF16PtrFromString("TaskbarCreated")
	if registered, _, _ := procRegisterWindowMessage.Call(uintptr(unsafe.Pointer(taskbarName))); registered != 0 {
		t.taskbarMessage = uint32(registered)
	}
	class := wndClassEx{
		CbSize:    uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:   syscall.NewCallback(windowProc),
		Instance:  instance,
		ClassName: className,
	}
	atom, _, callErr := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&class)))
	if atom == 0 {
		t.signalReady()
		return fmt.Errorf("register tray window class: %w", callErr)
	}
	defer procUnregisterClass.Call(uintptr(unsafe.Pointer(className)), instance)

	hwnd, _, callErr := procCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
		0,
		0, 0, 0, 0,
		0, 0, instance, 0,
	)
	if hwnd == 0 {
		t.signalReady()
		return fmt.Errorf("create tray window: %w", callErr)
	}
	t.setWindow(hwnd)
	traysMu.Lock()
	trays[hwnd] = t
	traysMu.Unlock()
	defer func() {
		traysMu.Lock()
		delete(trays, hwnd)
		traysMu.Unlock()
	}()
	procShowWindow.Call(hwnd, swHide)

	iconPathUTF16, _ := syscall.UTF16PtrFromString(iconPath)
	hicon, _, callErr := procLoadImage.Call(0, uintptr(unsafe.Pointer(iconPathUTF16)), imageIcon, 32, 32, lrLoadFromFile|lrDefaultSize)
	if hicon == 0 {
		t.signalReady()
		procDestroyWindow.Call(hwnd)
		return fmt.Errorf("load tray icon: %w", callErr)
	}
	t.setIcon(hicon)
	defer procDestroyIcon.Call(hicon)

	// Run-key applications can start while Explorer is still creating the
	// notification area. Treat that as a transient login race instead of
	// shutting down the whole media server after one failed API call.
	if err := retryOperation(initialIconAttempts, initialIconDelay, t.addIcon); err != nil {
		t.signalReady()
		procDestroyWindow.Call(hwnd)
		return err
	}
	defer t.removeIcon()
	t.signalReady()

	var msg message
	for {
		result, _, getErr := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(result) == -1 {
			return fmt.Errorf("get tray message: %w", getErr)
		}
		if result == 0 {
			return nil
		}
		procTranslateMsg.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMsg.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func (t *Tray) WaitReady() error {
	<-t.ready
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	if t.hwnd == 0 || !t.iconAdded {
		return errors.New("system tray did not initialize")
	}
	return nil
}

func (t *Tray) Close() {
	<-t.ready
	t.mutex.RLock()
	hwnd := t.hwnd
	t.mutex.RUnlock()
	if hwnd != 0 {
		procPostMessage.Call(hwnd, wmClose, 0, 0)
	}
	<-t.done
}

func (t *Tray) Notify(title, body string, isError bool) {
	t.mutex.RLock()
	hwnd := t.hwnd
	hicon := t.hicon
	added := t.iconAdded
	t.mutex.RUnlock()
	if hwnd == 0 || !added {
		return
	}
	data := notifyIconData{
		CbSize:    uint32(unsafe.Sizeof(notifyIconData{})),
		HWnd:      hwnd,
		ID:        1,
		Flags:     nifInfo,
		Icon:      hicon,
		InfoFlags: niifInfo,
	}
	if isError {
		data.InfoFlags = niifError
	}
	copyUTF16(data.InfoTitle[:], title)
	copyUTF16(data.Info[:], body)
	procShellNotifyIcon.Call(nimModify, uintptr(unsafe.Pointer(&data)))
}

func (t *Tray) addIcon() error {
	t.mutex.RLock()
	hwnd := t.hwnd
	hicon := t.hicon
	t.mutex.RUnlock()
	data := notifyIconData{
		CbSize:          uint32(unsafe.Sizeof(notifyIconData{})),
		HWnd:            hwnd,
		ID:              1,
		Flags:           nifMessage | nifIcon | nifTip,
		CallbackMessage: trayMessage,
		Icon:            hicon,
	}
	copyUTF16(data.Tip[:], "KoduMedia")
	result, _, callErr := procShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&data)))
	if result == 0 {
		return fmt.Errorf("add system tray icon: %w", callErr)
	}
	t.mutex.Lock()
	t.iconAdded = true
	t.mutex.Unlock()
	return nil
}

func (t *Tray) removeIcon() {
	t.mutex.Lock()
	if !t.iconAdded || t.hwnd == 0 {
		t.mutex.Unlock()
		return
	}
	data := notifyIconData{CbSize: uint32(unsafe.Sizeof(notifyIconData{})), HWnd: t.hwnd, ID: 1}
	t.iconAdded = false
	t.mutex.Unlock()
	procShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&data)))
}

func (t *Tray) showMenu() {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer procDestroyMenu.Call(menu)
	appendMenu(menu, mfString, menuOpenHome, "Ava Kodu")
	appendMenu(menu, mfSeparator, 0, "")
	startupFlags := uintptr(mfString)
	if t.callbacks.StartupEnabled != nil && t.callbacks.StartupEnabled() {
		startupFlags |= mfChecked
	}
	appendMenu(menu, startupFlags, menuStartup, "Käivita Kodu sisselogimisel")
	appendMenu(menu, mfString, menuScanAll, "Skanni kõik meediakogud")
	appendMenu(menu, mfString, menuSettings, "Seaded")
	appendMenu(menu, mfSeparator, 0, "")
	appendMenu(menu, mfString, menuExit, "Sulge Kodu")
	procSetMenuDefault.Call(menu, menuOpenHome, 0)

	var cursor point
	if result, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&cursor))); result == 0 {
		return
	}
	t.mutex.RLock()
	hwnd := t.hwnd
	t.mutex.RUnlock()
	procSetForeground.Call(hwnd)
	command, _, _ := procTrackPopupMenu.Call(menu, tpmRightButton|tpmNonotify|tpmReturnCmd, uintptr(cursor.X), uintptr(cursor.Y), 0, hwnd, 0)
	procPostMessage.Call(hwnd, wmNull, 0, 0)
	if command == 0 {
		return
	}
	go t.dispatch(command)
}

func (t *Tray) dispatch(command uintptr) {
	switch command {
	case menuOpenHome:
		if t.callbacks.OpenHome != nil {
			t.callbacks.OpenHome()
		}
	case menuStartup:
		if t.callbacks.ToggleStartup != nil {
			t.callbacks.ToggleStartup()
		}
	case menuScanAll:
		if t.callbacks.ScanAll != nil {
			t.callbacks.ScanAll()
		}
	case menuSettings:
		if t.callbacks.OpenSettings != nil {
			t.callbacks.OpenSettings()
		}
	case menuExit:
		if t.callbacks.Exit != nil {
			t.callbacks.Exit()
		}
	}
}

func (t *Tray) ensureIconFile() (string, error) {
	directory := filepath.Join(t.dataDir, "assets")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", fmt.Errorf("create tray asset directory: %w", err)
	}
	path := filepath.Join(directory, "kodu-media.ico")
	if current, err := os.ReadFile(path); err == nil && string(current) == string(iconBytes) {
		return path, nil
	}
	temporary, err := os.CreateTemp(directory, ".kodu-media-icon-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create tray icon: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(iconBytes); err != nil {
		_ = temporary.Close()
		return "", fmt.Errorf("write tray icon: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("close tray icon: %w", err)
	}
	_ = os.Remove(path)
	if err := os.Rename(temporaryPath, path); err != nil {
		return "", fmt.Errorf("install tray icon: %w", err)
	}
	return path, nil
}

func (t *Tray) setWindow(hwnd uintptr) {
	t.mutex.Lock()
	t.hwnd = hwnd
	t.mutex.Unlock()
}

func (t *Tray) setIcon(icon uintptr) {
	t.mutex.Lock()
	t.hicon = icon
	t.mutex.Unlock()
}

func (t *Tray) signalReady() {
	t.readyOnce.Do(func() { close(t.ready) })
}

func (t *Tray) signalDone() {
	t.doneOnce.Do(func() { close(t.done) })
}

func appendMenu(menu, flags, id uintptr, label string) {
	var labelPtr *uint16
	if label != "" {
		labelPtr, _ = syscall.UTF16PtrFromString(label)
	}
	procAppendMenu.Call(menu, flags, id, uintptr(unsafe.Pointer(labelPtr)))
}

func copyUTF16(destination []uint16, value string) {
	encoded, _ := syscall.UTF16FromString(value)
	if len(encoded) > len(destination) {
		encoded = encoded[:len(destination)]
		encoded[len(encoded)-1] = 0
	}
	copy(destination, encoded)
}

func windowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	traysMu.RLock()
	tray := trays[hwnd]
	traysMu.RUnlock()
	if tray != nil && tray.taskbarMessage != 0 && msg == tray.taskbarMessage {
		tray.mutex.Lock()
		tray.iconAdded = false
		tray.mutex.Unlock()
		_ = tray.addIcon()
		return 0
	}
	switch msg {
	case trayMessage:
		if tray != nil {
			switch uint32(lParam) {
			case wmLButtonUp, wmRButtonUp, wmContextMenu:
				tray.showMenu()
				return 0
			}
		}
	case wmClose:
		procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		if tray != nil {
			tray.removeIcon()
		}
		procPostQuitMessage.Call(0)
		return 0
	}
	result, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return result
}
