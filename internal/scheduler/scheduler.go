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
var ErrWorkerRequired = errors.New("scheduler: worker is required")
var ErrJobNotRunning = errors.New("scheduler: job must be running")
var ErrInvalidFinalStatus = errors.New("scheduler: final status must be succeeded or failed")

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

func Finish(assignedWorker *worker.Worker, candidate *job.Job, finalStatus job.Status) error {
	if candidate == nil {
		return ErrJobRequired
	}

	if assignedWorker == nil {
		return ErrWorkerRequired
	}

	if candidate.Status() != job.StatusRunning {
		return ErrJobNotRunning
	}

	if finalStatus != job.StatusFailed && finalStatus != job.StatusSucceeded {
		return ErrInvalidFinalStatus
	}

	err := assignedWorker.Release(candidate.Resources())

	if err != nil {
		return err
	}

	err = candidate.TransitionTo(finalStatus)

	if err != nil {
		reserveErr := assignedWorker.Reserve(candidate.Resources())
		return errors.Join(err, reserveErr)
	}

	return nil
}
