package service

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

type UpdateService struct {
	installer   ports.UpdateInstaller
	requestStop func()
	stopOnce    sync.Once
}

func NewUpdateService(installer ports.UpdateInstaller, requestStop func()) *UpdateService {
	return &UpdateService{installer: installer, requestStop: requestStop}
}

func (s *UpdateService) Status() domain.UpdateStatus {
	if s == nil || s.installer == nil {
		return domain.UpdateStatus{State: domain.UpdateStateIdle, LocalOnly: true}
	}
	return s.installer.Status()
}

func (s *UpdateService) Install(ctx context.Context, packageReader io.Reader, packageName string) (domain.UpdateStatus, error) {
	if s == nil || s.installer == nil {
		return domain.UpdateStatus{}, fmt.Errorf("%w: update installer is not configured", domain.ErrUnavailable)
	}
	status, err := s.installer.Install(ctx, packageReader, packageName)
	if err != nil {
		return status, err
	}
	if s.requestStop != nil {
		time.AfterFunc(1200*time.Millisecond, func() {
			s.stopOnce.Do(s.requestStop)
		})
	}
	return status, nil
}
