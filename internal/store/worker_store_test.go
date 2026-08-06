package store

import (
	"errors"
	"testing"

	"github.com/sanketlengare/orchard/internal/worker"
)

func TestRegister(t *testing.T) {

	mwStore := NewMemoryWorkerStore()

	nilErr := mwStore.Register(nil)
	wantErr := ErrWorkerRequired
	if !errors.Is(nilErr, wantErr) {
		t.Fatalf("Create() error = %v, want %v", nilErr, wantErr)
	}

	candidate, err := worker.New("worker1", worker.Capacity{CPUCores: 2, MemoryMiB: 4096})

	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = mwStore.Register(candidate)

	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	duplicate, err := worker.New("worker1", worker.Capacity{CPUCores: 1, MemoryMiB: 1024})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = mwStore.Register(duplicate)
	if !errors.Is(err, ErrWorkerAlreadyExists) {
		t.Fatalf("Register() error = %v, want %v", err, ErrWorkerAlreadyExists)
	}

	stored, err := mwStore.Get("worker1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if stored != candidate {
		t.Fatalf("Get() job = %p, want original %p", stored, candidate)
	}
}

func TestGetWorkerStore(t *testing.T) {
	mwStore := NewMemoryWorkerStore()

	candidate, err := worker.New("worker1", worker.Capacity{CPUCores: 2, MemoryMiB: 4096})

	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	createErr := mwStore.Register(candidate)

	if createErr != nil {
		t.Fatalf("Create() error = %v", createErr)
	}

	w, err := mwStore.Get("worker2")
	wantErr := ErrWorkerNotFound

	if !errors.Is(err, wantErr) {
		t.Fatalf("Get() error = %v, want %v", err, wantErr)
	}

	if w != nil {
		t.Fatalf("Get() recieved %v, want %v", w, nil)
	}

	w, err = mwStore.Get(candidate.ID())

	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if w != candidate {
		t.Fatalf("Get() job = %p, want original %p", w, candidate)
	}

}

func TestListWorkers(t *testing.T) {
	mwStore := NewMemoryWorkerStore()

	candidate1, _ := worker.New("worker1", worker.Capacity{CPUCores: 2, MemoryMiB: 4096})

	candidate2, _ := worker.New("worker2", worker.Capacity{CPUCores: 4, MemoryMiB: 8192})

	workers, err := mwStore.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if workers == nil {
		t.Fatal("List() returned nil slice, want non-nil empty slice")
	}

	if len(workers) != 0 {
		t.Fatalf("List() length = %d, want 0", len(workers))
	}

	mwStore.Register(candidate2)

	workers, err = mwStore.List()

	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(workers) != 1 {
		t.Fatalf("List() recieved len = %v, want len = 1", len(workers))
	}

	if workers[0] != candidate2 {
		t.Fatalf("List() recieved %v, want %v", workers[0].ID(), candidate2.ID())
	}

	mwStore.Register(candidate1)

	workers, err = mwStore.List()

	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(workers) != 2 {
		t.Fatalf("List() recieved len = %v, want len = 2", len(workers))
	}

	if workers[0].ID() >= workers[1].ID() {
		t.Fatalf("List() recieved order of ID's %v, %v, want %v, %v", candidate2.ID(), candidate1.ID(), candidate1.ID(), candidate2.ID())
	}
}
func TestModifyingListWorkers(t *testing.T) {

	mwStore := NewMemoryWorkerStore()

	candidate1, _ := worker.New("worker1", worker.Capacity{CPUCores: 2, MemoryMiB: 4096})

	mwStore.Register(candidate1)

	workers, err := mwStore.List()

	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if workers[0] != candidate1 {
		t.Fatalf("List() recieved %v, want %v", workers[0].ID(), candidate1.ID())
	}

	candidate2, err := worker.New("worker2", worker.Capacity{CPUCores: 4, MemoryMiB: 8192})

	workers[0] = candidate2

	workers, err = mwStore.List()

	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if workers[0] != candidate1 {
		t.Fatalf("List() recieved %v, want %v", workers[0].ID(), candidate1.ID())
	}

}
