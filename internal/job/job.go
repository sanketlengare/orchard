package job

import (
	"errors"
	"slices"
	"strings"
)

var ErrIDRequired = errors.New("job: id is required")
var ErrCommandRequired = errors.New("job: command is required")
var ErrInvalidTransition = errors.New("job: invalid status transition")

type Job struct {
	id        string
	command   []string
	resources ResourceRequest
	status    Status
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
	return j.status
}

func (j *Job) TransitionTo(next Status) error {
	if !j.status.CanTransitionTo(next) {
		return ErrInvalidTransition
	}
	j.status = next
	return nil
}

type ResourceRequest struct {
	CPUCores  int
	MemoryMiB int
}

var ErrCPUCoresRequired = errors.New("job: cpu cores must be greater than zero")

var ErrMemoryMiBRequired = errors.New("job: memory MiB must be greater than zero")
