//go:build !windows

package selfupdate

import (
	"context"
	"fmt"
	"io"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

type Manager struct{ currentVersion string }

func New(_, currentVersion, _ string) (*Manager, error) {
	return &Manager{currentVersion: currentVersion}, nil
}

func (m *Manager) Status() domain.UpdateStatus {
	return domain.UpdateStatus{State: domain.UpdateStateIdle, CurrentVersion: m.currentVersion, LocalOnly: true}
}

func (m *Manager) Install(context.Context, io.Reader, string) (domain.UpdateStatus, error) {
	return m.Status(), fmt.Errorf("%w: rakendusesisene uuendamine on toetatud ainult Windowsis", domain.ErrUnsupported)
}
