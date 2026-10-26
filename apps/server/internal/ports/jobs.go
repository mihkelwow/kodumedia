package ports

import "github.com/mihkelwow/kodumedia/internal/domain"

type JobProgress func(progress int, message string)

type JobRunner interface {
	Start(jobType string, run func(update JobProgress) error) domain.Job
	Get(id string) (domain.Job, bool)
}
