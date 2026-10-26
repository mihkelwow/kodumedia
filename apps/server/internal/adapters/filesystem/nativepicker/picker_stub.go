//go:build !windows

package nativepicker

import (
	"context"
	"fmt"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

type Picker struct{}

func New() *Picker { return &Picker{} }

func (p *Picker) Pick(context.Context, domain.FilePickerRequest) (string, bool, error) {
	return "", false, fmt.Errorf("%w: native picker is available only on Windows", domain.ErrUnavailable)
}

var _ ports.FilePicker = (*Picker)(nil)
