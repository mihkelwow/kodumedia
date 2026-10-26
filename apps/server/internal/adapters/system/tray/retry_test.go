package tray

import (
	"errors"
	"testing"
)

func TestRetryOperationRecoversFromTemporaryShellFailure(t *testing.T) {
	attempts := 0
	err := retryOperation(4, 0, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("taskbar is not ready")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("retryOperation: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestRetryOperationReturnsLastErrorAfterExhaustion(t *testing.T) {
	first := errors.New("first")
	last := errors.New("last")
	attempts := 0
	err := retryOperation(2, 0, func() error {
		attempts++
		if attempts == 1 {
			return first
		}
		return last
	})
	if !errors.Is(err, last) {
		t.Fatalf("retryOperation error = %v, want last error", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestRetryOperationRejectsNilOperation(t *testing.T) {
	if err := retryOperation(1, 0, nil); !errors.Is(err, errMissingRetryOperation) {
		t.Fatalf("retryOperation error = %v", err)
	}
}
