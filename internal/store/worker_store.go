package store

import (
	"errors"
	"sort"
	"sync"

	"github.com/sanketlengare/orchard/internal/worker"
)

var ErrWorkerRequired = errors.New("store: worker is required")
var ErrWorkerAlreadyExists = errors.New("store: worker already exists")
var ErrWorkerNotFound = errors.New("store: worker not found")

type MemoryWorkerStore struct {
	mu      sync.RWMutex
	workers map[string]*worker.Worker
}

func NewMemoryWorkerStore() *MemoryWorkerStore {
	return &MemoryWorkerStore{
		workers: make(map[string]*worker.Worker),
	}
}

func (s *MemoryWorkerStore) Register(candidate *worker.Worker) error {
	if candidate == nil {
		return ErrWorkerRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.workers[candidate.ID()]; ok {
		return ErrWorkerAlreadyExists
	}

	s.workers[candidate.ID()] = candidate
	return nil
}

func (s *MemoryWorkerStore) Get(id string) (*worker.Worker, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	worker, ok := s.workers[id]

	if !ok {
		return nil, ErrWorkerNotFound
	}

	return worker, nil
}

func (s *MemoryWorkerStore) List() ([]*worker.Worker, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.workers) == 0 {
		return []*worker.Worker{}, nil
	}

	workers := make([]*worker.Worker, 0, len(s.workers))

	for _, w := range s.workers {
		workers = append(workers, w)
	}

	sort.Slice(workers, func(i, j int) bool {
		return workers[i].ID() < workers[j].ID()
	})

	return workers, nil
}
