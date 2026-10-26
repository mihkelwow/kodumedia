//go:build !windows

package childprocess

import "os/exec"

// Hide is a no-op outside Windows. Keeping the platform detail behind this
// small adapter lets the media services remain portable and easy to test.
func Hide(command *exec.Cmd) {
	_ = command
}
