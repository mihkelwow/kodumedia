//go:build windows

package childprocess

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// Hide prevents console-based helper programs such as FFmpeg and FFprobe from
// opening a transient terminal window when KoduMedia runs as a Windows GUI app.
func Hide(command *exec.Cmd) {
	if command == nil {
		return
	}
	command.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
}
