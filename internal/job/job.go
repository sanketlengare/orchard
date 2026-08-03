package job

import (
	"errors"
	"slices"
	"strings"
)

var ErrIDRequired = errors.New("job: id is required")
var ErrCommandRequired = errors.New("job: command is required")

type Job struct {
	id      string
	command []string
	status  Status
}

func New(id string, command []string) (*Job, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrIDRequired
	}

	if len(command) == 0 || strings.TrimSpace(command[0]) == "" {
		return nil, ErrCommandRequired
	}

	return &Job{
		id:      id,
		command: slices.Clone(command),
		status:  StatusPending,
	}, nil
}

func (j *Job) ID() string {
	return j.id
}

func (j *Job) Command() []string {
	return slices.Clone(j.command)
}

func (j *Job) Status() Status {
	return j.status
}
