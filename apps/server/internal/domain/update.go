package domain

import "time"

type UpdateState string

const (
	UpdateStateIdle       UpdateState = "idle"
	UpdateStatePreparing  UpdateState = "preparing"
	UpdateStateInstalling UpdateState = "installing"
	UpdateStateRestarting UpdateState = "restarting"
	UpdateStateSucceeded  UpdateState = "succeeded"
	UpdateStateFailed     UpdateState = "failed"
)

type UpdateStatus struct {
	State           UpdateState `json:"state"`
	CurrentVersion  string      `json:"currentVersion"`
	TargetVersion   string      `json:"targetVersion,omitempty"`
	PackageName     string      `json:"packageName,omitempty"`
	Message         string      `json:"message,omitempty"`
	Error           string      `json:"error,omitempty"`
	LocalOnly       bool        `json:"localOnly"`
	StartedAt       *time.Time  `json:"startedAt,omitempty"`
	UpdatedAt       *time.Time  `json:"updatedAt,omitempty"`
	CompletedAt     *time.Time  `json:"completedAt,omitempty"`
	RollbackApplied bool        `json:"rollbackApplied,omitempty"`
}
