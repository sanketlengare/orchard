package scheduler

import (
	"errors"

	"github.com/sanketlengare/orchard/internal/job"
	"github.com/sanketlengare/orchard/internal/worker"
)

var ErrInvalidResourceRequest = errors.New("scheduler: invalid resource request")
var ErrNoSuitableWorker = errors.New("scheduler: no suitable worker")
var ErrJobRequired = errors.New("scheduler: job is required")
var ErrJobNotPending = errors.New("scheduler: job must be pending")

func Select(workers []*worker.Worker, request job.ResourceRequest) (*worker.Worker, error) {
	if request.CPUCores <= 0 || request.MemoryMiB <= 0 {
		return nil, ErrInvalidResourceRequest
	}

	for _, w := range workers {
		if w != nil && w.CanRun(request) {
			return w, nil
		}
	}

	return nil, ErrNoSuitableWorker
}

func Assign(workers []*worker.Worker, candidate *job.Job) (*worker.Worker, error) {
	if candidate == nil {
		return nil, ErrJobRequired
	}

	if candidate.Status() != job.StatusPending {
		return nil, ErrJobNotPending
	}

	resources := candidate.Resources()

	selectedWorker, err := Select(workers, resources)
	if err != nil {
		return nil, err
	}

	if err := selectedWorker.Reserve(resources); err != nil {
		return nil, err
	}

	if err := candidate.TransitionTo(job.StatusRunning); err != nil {
		releaseErr := selectedWorker.Release(resources)
		return nil, errors.Join(err, releaseErr)
	}

	return selectedWorker, nil

}
