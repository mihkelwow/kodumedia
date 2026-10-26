//go:build !windows

package updater

import "os/exec"

func configureChildProcess(_ *exec.Cmd) {}
