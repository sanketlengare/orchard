package worker

import (
	"errors"
	"strings"

	"github.com/sanketlengare/orchard/internal/job"
)

type Capacity struct {
	CPUCores  int
	MemoryMiB int
}

var ErrIDRequired = errors.New("worker: id is required")
var ErrCPUCoresRequired = errors.New("worker: cpu cores must be greater than zero")
var ErrMemoryMiBRequired = errors.New("worker: memory MiB must be greater than zero")

type Worker struct {
	id        string
	total     Capacity
	available Capacity
}

func New(id string, total Capacity) (*Worker, error) {
	id = strings.TrimSpace(id)

	if id == "" {
		return nil, ErrIDRequired
	}

	if total.CPUCores <= 0 {
		return nil, ErrCPUCoresRequired
	}

	if total.MemoryMiB <= 0 {
		return nil, ErrMemoryMiBRequired
	}

	return &Worker{
		id:        id,
		total:     total,
		available: total,
	}, nil
}

func (w *Worker) ID() string {
	return w.id
}

func (w *Worker) TotalCapacity() Capacity {
	return w.total
}

func (w *Worker) AvailableCapacity() Capacity {
	return w.available
}

func (w *Worker) CanRun(request job.ResourceRequest) bool {
	return request.CPUCores > 0 && request.MemoryMiB > 0 && request.CPUCores <= w.available.CPUCores && request.MemoryMiB <= w.available.MemoryMiB
}
