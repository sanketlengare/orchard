package store

import (
	"errors"
	"sort"
	"sync"

	"github.com/sanketlengare/orchard/internal/job"
)

var ErrJobRequired = errors.New("store: job is required")
var ErrJobAlreadyExists = errors.New("store: job already exists")
var ErrJobNotFound = errors.New("store: job not found")

type MemoryJobStore struct {
	mu   sync.RWMutex
	jobs map[string]*job.Job
}

func NewMemoryJobStore() *MemoryJobStore {
	return &MemoryJobStore{
		jobs: make(map[string]*job.Job),
	}
}

func (s *MemoryJobStore) Create(candidate *job.Job) error {
	if candidate == nil {
		return ErrJobRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.jobs[candidate.ID()]; ok {
		return ErrJobAlreadyExists
	}

	s.jobs[candidate.ID()] = candidate
	return nil
}

func (s *MemoryJobStore) Get(id string) (*job.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.jobs[id]

	if !ok {
		return nil, ErrJobNotFound
	}

	return job, nil
}

func (s *MemoryJobStore) List() ([]*job.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.jobs) == 0 {
		return []*job.Job{}, nil
	}

	jobs := make([]*job.Job, 0, len(s.jobs))

	for _, j := range s.jobs {
		jobs = append(jobs, j)
	}

	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].ID() < jobs[j].ID()
	})

	return jobs, nil
}
