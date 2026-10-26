//go:build windows

package updater

import (
	"os/exec"
	"syscall"
)

const (
	childCreateNewProcessGroup = 0x00000200
	childCreateNoWindow        = 0x08000000
)

func configureChildProcess(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: childCreateNewProcessGroup | childCreateNoWindow}
}
