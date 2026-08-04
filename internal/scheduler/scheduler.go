package scheduler

import (
	"errors"

	"github.com/sanketlengare/orchard/internal/job"
	"github.com/sanketlengare/orchard/internal/worker"
)

var ErrInvalidResourceRequest = errors.New("scheduler: invalid resource request")
var ErrNoSuitableWorker = errors.New("scheduler: no suitable worker")

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
