package worker

import (
	"errors"
	"strings"
	"sync"

	"github.com/sanketlengare/orchard/internal/job"
)

var ErrIDRequired = errors.New("worker: id is required")
var ErrCPUCoresRequired = errors.New("worker: cpu cores must be greater than zero")
var ErrMemoryMiBRequired = errors.New("worker: memory MiB must be greater than zero")
var ErrInvalidResourceRequest = errors.New("worker: resource request must be greater than zero")
var ErrInsufficientCapacity = errors.New("worker: insufficient available capacity")
var ErrReleaseExceedsUsage = errors.New("worker: release exceeds used capacity")

type Capacity struct {
	CPUCores  int
	MemoryMiB int
}

type Worker struct {
	id        string
	total     Capacity
	available Capacity
	mu        sync.RWMutex
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
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.available
}

// canRunLocked requires the caller to hold w.mu.
func (w *Worker) canRunLocked(request job.ResourceRequest) bool {

	return request.CPUCores > 0 && request.MemoryMiB > 0 && request.CPUCores <= w.available.CPUCores && request.MemoryMiB <= w.available.MemoryMiB
}

func (w *Worker) CanRun(request job.ResourceRequest) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.canRunLocked(request)
}

func (w *Worker) Reserve(request job.ResourceRequest) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !validRequest(request) {
		return ErrInvalidResourceRequest
	}

	if !w.canRunLocked(request) {
		return ErrInsufficientCapacity
	}

	w.available.CPUCores -= request.CPUCores
	w.available.MemoryMiB -= request.MemoryMiB

	return nil
}

func (w *Worker) Release(request job.ResourceRequest) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !validRequest(request) {
		return ErrInvalidResourceRequest
	}

	usedCPU := w.total.CPUCores - w.available.CPUCores
	usedMemory := w.total.MemoryMiB - w.available.MemoryMiB

	if request.CPUCores > usedCPU || request.MemoryMiB > usedMemory {
		return ErrReleaseExceedsUsage
	}

	w.available.CPUCores += request.CPUCores
	w.available.MemoryMiB += request.MemoryMiB

	return nil
}

func validRequest(request job.ResourceRequest) bool {
	return request.CPUCores > 0 && request.MemoryMiB > 0
}
