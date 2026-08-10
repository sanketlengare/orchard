package controlplane

import (
	"errors"

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

type Service struct {
	jobStore    JobStore
	workerStore WorkerStore
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

	return assignedWorker, nil
}
