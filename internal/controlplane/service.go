package controlplane

import (
	"errors"
	"sync"

	"github.com/sanketlengare/orchard/internal/job"
	"github.com/sanketlengare/orchard/internal/scheduler"
	"github.com/sanketlengare/orchard/internal/worker"
)

type JobStore interface {
	Create(*job.Job) error
	Get(string) (*job.Job, error)
	List() ([]*job.Job, error)
}

type WorkerStore interface {
	Register(*worker.Worker) error
	Get(string) (*worker.Worker, error)
	List() ([]*worker.Worker, error)
}

var ErrJobStoreRequired = errors.New("controlplane: job store is required")
var ErrWorkerStoreRequired = errors.New("controlplane: worker store is required")
var ErrAssignmentNotFound = errors.New("controlplane: assignment not found")

type Service struct {
	mu          sync.RWMutex
	jobStore    JobStore
	workerStore WorkerStore
	assignments map[string]string
}

func NewService(jobStore JobStore, workerStore WorkerStore) (*Service, error) {
	if jobStore == nil {
		return nil, ErrJobStoreRequired
	}

	if workerStore == nil {
		return nil, ErrWorkerStoreRequired
	}

	return &Service{
		jobStore:    jobStore,
		workerStore: workerStore,
		assignments: make(map[string]string),
	}, nil
}

func (s *Service) SubmitJob(id string, command []string, resources job.ResourceRequest) (*job.Job, error) {
	created, newErr := job.New(id, command, resources)

	if newErr != nil {
		return nil, newErr
	}

	createErr := s.jobStore.Create(created)

	if createErr != nil {
		return nil, createErr
	}

	return created, nil
}

func (s *Service) RegisterWorker(id string, capacity worker.Capacity) (*worker.Worker, error) {
	created, newErr := worker.New(id, capacity)

	if newErr != nil {
		return nil, newErr
	}

	regErr := s.workerStore.Register(created)

	if regErr != nil {
		return nil, regErr
	}

	return created, nil
}

func (s *Service) ScheduleJob(jobID string) (*worker.Worker, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	j, getErr := s.jobStore.Get(jobID)

	if getErr != nil {
		return nil, getErr
	}

	workersList, err := s.workerStore.List()

	if err != nil {
		return nil, err
	}

	assignedWorker, aErr := scheduler.Assign(workersList, j)

	if aErr != nil {
		return nil, aErr
	}

	s.assignments[jobID] = assignedWorker.ID()

	return assignedWorker, nil
}

func (s *Service) AssignedWorker(jobID string) (*worker.Worker, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.assignments[jobID]; !ok {
		return nil, ErrAssignmentNotFound
	}

	workerID := s.assignments[jobID]

	w, err := s.workerStore.Get(workerID)

	if err != nil {
		return nil, err
	}

	return w, nil
}

func (s *Service) FinishJob(jobID string, finalStatus job.Status) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	candidate, err := s.jobStore.Get(jobID)
	if err != nil {
		return err
	}

	workerID, ok := s.assignments[jobID]
	if !ok {
		return ErrAssignmentNotFound
	}

	assignedWorker, err := s.workerStore.Get(workerID)
	if err != nil {
		return err
	}

	if err := scheduler.Finish(
		assignedWorker,
		candidate,
		finalStatus,
	); err != nil {
		return err
	}

	delete(s.assignments, jobID)

	return nil
}
