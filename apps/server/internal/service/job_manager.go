package service

import (
	"sync"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

type JobManager struct {
	mu   sync.RWMutex
	jobs map[string]domain.Job
	now  func() time.Time
}

func NewJobManager() *JobManager {
	return &JobManager{jobs: map[string]domain.Job{}, now: time.Now}
}

func (m *JobManager) Start(jobType string, run func(update ports.JobProgress) error) domain.Job {
	job := domain.Job{
		ID:        randomID("job"),
		Type:      jobType,
		Status:    domain.JobStatusQueued,
		Progress:  0,
		CreatedAt: m.now().UTC(),
	}
	m.mu.Lock()
	m.jobs[job.ID] = job
	m.mu.Unlock()

	go m.execute(job.ID, run)
	return job
}

func (m *JobManager) Get(id string) (domain.Job, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, ok := m.jobs[id]
	return job, ok
}

func (m *JobManager) execute(id string, run func(update ports.JobProgress) error) {
	started := m.now().UTC()
	m.update(id, func(job *domain.Job) {
		job.Status = domain.JobStatusRunning
		job.StartedAt = &started
		job.Progress = 1
	})

	err := run(func(progress int, message string) {
		if progress < 1 {
			progress = 1
		}
		if progress > 99 {
			progress = 99
		}
		m.update(id, func(job *domain.Job) {
			job.Progress = progress
			job.Message = message
		})
	})

	completed := m.now().UTC()
	m.update(id, func(job *domain.Job) {
		job.CompletedAt = &completed
		if err != nil {
			job.Status = domain.JobStatusFailed
			job.Error = err.Error()
			return
		}
		job.Status = domain.JobStatusSucceeded
		job.Progress = 100
		job.Message = "Valmis"
	})
}

func (m *JobManager) update(id string, mutate func(*domain.Job)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return
	}
	mutate(&job)
	m.jobs[id] = job
}

var _ ports.JobRunner = (*JobManager)(nil)
