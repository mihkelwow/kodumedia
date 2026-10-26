package domain

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrPending      = errors.New("pending")
	ErrConflict     = errors.New("conflict")
	ErrForbidden    = errors.New("forbidden")
	ErrInvalidInput = errors.New("invalid input")
	ErrUnsupported  = errors.New("unsupported")
	ErrUnavailable  = errors.New("unavailable")
)
