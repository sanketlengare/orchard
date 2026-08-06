package store

import (
	"errors"
	"testing"

	"github.com/sanketlengare/orchard/internal/job"
)

func TestCreate(t *testing.T) {

	mjStore := NewMemoryJobStore()

	nilErr := mjStore.Create(nil)
	wantErr := ErrJobRequired
	if !errors.Is(nilErr, wantErr) {
		t.Fatalf("Create() error = %v, want %v", nilErr, wantErr)
	}

	candidate, err := job.New("job1", []string{"echo", "hello"}, job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096})

	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = mjStore.Create(candidate)

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	duplicate, err := job.New(
		"job1",
		[]string{"printf", "different"},
		job.ResourceRequest{CPUCores: 1, MemoryMiB: 1024},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = mjStore.Create(duplicate)
	if !errors.Is(err, ErrJobAlreadyExists) {
		t.Fatalf("Create() error = %v, want %v", err, ErrJobAlreadyExists)
	}

	stored, err := mjStore.Get("job1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if stored != candidate {
		t.Fatalf("Get() job = %p, want original %p", stored, candidate)
	}
}

func TestGet(t *testing.T) {
	mjStore := NewMemoryJobStore()

	candidate, jobNewErr := job.New("job1", []string{"echo", "hello"}, job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096})

	if jobNewErr != nil {
		t.Fatalf("New() error = %v", jobNewErr)
	}

	createErr := mjStore.Create(candidate)

	if createErr != nil {
		t.Fatalf("Create() error = %v", createErr)
	}

	j, err := mjStore.Get("job2")
	wantErr := ErrJobNotFound

	if !errors.Is(err, wantErr) {
		t.Fatalf("Create() error = %v, want %v", err, wantErr)
	}

	if j != nil {
		t.Fatalf("Get() recieved %v, want %v", j, nil)
	}

	j, err = mjStore.Get(candidate.ID())

	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if j != candidate {
		t.Fatalf("Get() job = %p, want original %p", j, candidate)
	}

}

func TestList(t *testing.T) {
	mjStore := NewMemoryJobStore()

	candidate1, _ := job.New("job1", []string{"echo", "hello"}, job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096})

	candidate2, _ := job.New("job2", []string{"echo", " "}, job.ResourceRequest{CPUCores: 4, MemoryMiB: 8192})

	jobs, err := mjStore.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if jobs == nil {
		t.Fatal("List() returned nil slice, want non-nil empty slice")
	}

	if len(jobs) != 0 {
		t.Fatalf("List() length = %d, want 0", len(jobs))
	}

	mjStore.Create(candidate2)

	jobs, err = mjStore.List()

	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("List() recieved len = %v, want len = 1", len(jobs))
	}

	if jobs[0] != candidate2 {
		t.Fatalf("List() recieved %v, want %v", jobs[0].ID(), candidate2.ID())
	}

	mjStore.Create(candidate1)

	jobs, err = mjStore.List()

	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(jobs) != 2 {
		t.Fatalf("List() recieved len = %v, want len = 2", len(jobs))
	}

	if jobs[0].ID() >= jobs[1].ID() {
		t.Fatalf("List() recieved order of ID's %v, %v, want %v, %v", candidate2.ID(), candidate1.ID(), candidate1.ID(), candidate2.ID())
	}
}
func TestModifyingList(t *testing.T) {

	mjStore := NewMemoryJobStore()

	candidate1, _ := job.New("job1", []string{"echo", "hello"}, job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096})

	mjStore.Create(candidate1)

	jobs, err := mjStore.List()

	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if jobs[0] != candidate1 {
		t.Fatalf("List() recieved %v, want %v", jobs[0].ID(), candidate1.ID())
	}

	candidate2, _ := job.New("job2", []string{"echo", " "}, job.ResourceRequest{CPUCores: 4, MemoryMiB: 8192})

	jobs[0] = candidate2

	jobs, err = mjStore.List()

	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if jobs[0] != candidate1 {
		t.Fatalf("List() recieved %v, want %v", jobs[0].ID(), candidate1.ID())
	}

}
