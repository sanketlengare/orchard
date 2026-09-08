package job

import (
	"errors"
	"slices"
	"strings"
	"sync"
)

var ErrIDRequired = errors.New("job: id is required")
var ErrCommandRequired = errors.New("job: command is required")
var ErrInvalidTransition = errors.New("job: invalid status transition")
var ErrCPUCoresRequired = errors.New("job: cpu cores must be greater than zero")
var ErrMemoryMiBRequired = errors.New("job: memory MiB must be greater than zero")

type ResourceRequest struct {
	CPUCores  int
	MemoryMiB int
}

type Job struct {
	id        string
	command   []string
	resources ResourceRequest
	status    Status
	mu        sync.RWMutex
}

func New(id string, command []string, resources ResourceRequest) (*Job, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrIDRequired
	}

	if len(command) == 0 || strings.TrimSpace(command[0]) == "" {
		return nil, ErrCommandRequired
	}

	if resources.CPUCores <= 0 {
		return nil, ErrCPUCoresRequired
	}

	if resources.MemoryMiB <= 0 {
		return nil, ErrMemoryMiBRequired
	}

	return &Job{
		id:        id,
		command:   slices.Clone(command),
		resources: resources,
		status:    StatusPending,
	}, nil
}

func (j *Job) ID() string {
	return j.id
}

func (j *Job) Command() []string {
	return slices.Clone(j.command)
}

func (j *Job) Resources() ResourceRequest {
	return j.resources
}

func (j *Job) Status() Status {
	j.mu.RLock()
	defer j.mu.RUnlock()

	return j.status
}

func (j *Job) TransitionTo(next Status) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if !j.status.CanTransitionTo(next) {
		return ErrInvalidTransition
	}
	j.status = next
	return nil
}
