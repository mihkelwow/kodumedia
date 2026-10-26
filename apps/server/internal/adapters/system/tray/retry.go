package tray

import (
	"errors"
	"time"
)

var errMissingRetryOperation = errors.New("retry operation is nil")

// retryOperation retries short-lived Windows shell integration failures. At
// login Explorer can create the taskbar after Run-key applications have
// already started, so a failed first Shell_NotifyIcon call is not final.
func retryOperation(attempts int, delay time.Duration, operation func() error) error {
	if operation == nil {
		return errMissingRetryOperation
	}
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if err := operation(); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if attempt+1 < attempts && delay > 0 {
			time.Sleep(delay)
		}
	}
	return lastErr
}
