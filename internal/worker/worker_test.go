package worker

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/sanketlengare/orchard/internal/job"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		total   Capacity
		wantErr error
	}{
		{
			name:    "valid worker",
			id:      "worker1",
			total:   Capacity{CPUCores: 4, MemoryMiB: 8192},
			wantErr: nil,
		},
		{
			name:    "missing id",
			id:      "",
			total:   Capacity{CPUCores: 4, MemoryMiB: 8192},
			wantErr: ErrIDRequired,
		},
		{
			name:    "zero cpu cores",
			id:      "worker2",
			total:   Capacity{CPUCores: 0, MemoryMiB: 8192},
			wantErr: ErrCPUCoresRequired,
		},
		{
			name:    "zero memory",
			id:      "worker3",
			total:   Capacity{CPUCores: 4, MemoryMiB: 0},
			wantErr: ErrMemoryMiBRequired,
		},

		{
			name:    "negative cpu cores",
			id:      "worker4",
			total:   Capacity{CPUCores: -2, MemoryMiB: 8192},
			wantErr: ErrCPUCoresRequired,
		},
		{
			name:    "negative memory",
			id:      "worker5",
			total:   Capacity{CPUCores: 4, MemoryMiB: -1024},
			wantErr: ErrMemoryMiBRequired,
		},
		{
			name:    "whitespace id",
			id:      "   ",
			total:   Capacity{CPUCores: 4, MemoryMiB: 8192},
			wantErr: ErrIDRequired,
		},
		{
			name:    "whitespace around id",
			id:      "  worker6  ",
			total:   Capacity{CPUCores: 4, MemoryMiB: 8192},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created, err := New(tt.id, tt.total)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				if created != nil {
					t.Errorf("New() worker = %v, want nil", created)
				}
				return
			}

			if created == nil {
				t.Fatal("New() worker = nil, want a worker")
			}

			if got := created.ID(); got != strings.TrimSpace(tt.id) {
				t.Errorf("ID() = %q, want %q", got, strings.TrimSpace(tt.id))
			}

			if got := created.TotalCapacity(); got != tt.total {
				t.Errorf("TotalCapacity() = %v, want %v", got, tt.total)
			}

			if got := created.AvailableCapacity(); got != tt.total {
				t.Errorf("AvailableCapacity() = %v, want %v", got, tt.total)
			}

		})
	}
}

func TestWorkerCanRun(t *testing.T) {
	tests := []struct {
		name     string
		worker   Capacity
		request  job.ResourceRequest
		expected bool
	}{
		{
			name:     "can run",
			worker:   Capacity{CPUCores: 4, MemoryMiB: 8192},
			request:  job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096},
			expected: true,
		},
		{
			name:     "can run, exact fit",
			worker:   Capacity{CPUCores: 4, MemoryMiB: 8192},
			request:  job.ResourceRequest{CPUCores: 4, MemoryMiB: 8192},
			expected: true,
		},
		{
			name:     "cannot run due to CPU",
			worker:   Capacity{CPUCores: 4, MemoryMiB: 8192},
			request:  job.ResourceRequest{CPUCores: 5, MemoryMiB: 4096},
			expected: false,
		},
		{
			name:     "cannot run due to memory",
			worker:   Capacity{CPUCores: 4, MemoryMiB: 8192},
			request:  job.ResourceRequest{CPUCores: 2, MemoryMiB: 16384},
			expected: false,
		},
		{
			name:     "cannot run due to zero CPU",
			worker:   Capacity{CPUCores: 4, MemoryMiB: 8192},
			request:  job.ResourceRequest{CPUCores: 0, MemoryMiB: 4096},
			expected: false,
		},
		{
			name:     "cannot run due to zero memory",
			worker:   Capacity{CPUCores: 4, MemoryMiB: 8192},
			request:  job.ResourceRequest{CPUCores: 2, MemoryMiB: 0},
			expected: false,
		},
		{
			name:     "both CPU and memory exceed",
			worker:   Capacity{CPUCores: 4, MemoryMiB: 8192},
			request:  job.ResourceRequest{CPUCores: 5, MemoryMiB: 16384},
			expected: false,
		},
		{
			name:     "negative CPU cores",
			worker:   Capacity{CPUCores: 4, MemoryMiB: 8192},
			request:  job.ResourceRequest{CPUCores: -2, MemoryMiB: 4096},
			expected: false,
		},
		{
			name:     "negative memory",
			worker:   Capacity{CPUCores: 4, MemoryMiB: 8192},
			request:  job.ResourceRequest{CPUCores: 2, MemoryMiB: -4096},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := New("worker1", tt.worker)
			if err != nil {
				t.Fatalf("New() error = %v, wantErr %v", err, nil)
			}

			result := w.CanRun(tt.request)
			if result != tt.expected {
				t.Errorf("CanRun() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestWorkerCapacityReturnsCopy(t *testing.T) {
	w, err := New("worker1", Capacity{CPUCores: 4, MemoryMiB: 8192})
	if err != nil {
		t.Fatalf("New() error = %v, wantErr %v", err, nil)
	}

	total := w.TotalCapacity()
	available := w.AvailableCapacity()

	// Modify the returned values
	total.CPUCores = 0
	total.MemoryMiB = 0
	available.CPUCores = 0
	available.MemoryMiB = 0

	if total != (Capacity{}) || available != (Capacity{}) {
		t.Fatalf("mutated copies = total %v, available %v, want zero values", total, available)
	}

	// Check if the original values in the worker remain unchanged
	if got := w.TotalCapacity(); got.CPUCores != 4 || got.MemoryMiB != 8192 {
		t.Errorf("TotalCapacity() = %v, want {CPUCores: 4, MemoryMiB: 8192}", got)
	}

	if got := w.AvailableCapacity(); got.CPUCores != 4 || got.MemoryMiB != 8192 {
		t.Errorf("AvailableCapacity() = %v, want {CPUCores: 4, MemoryMiB: 8192}", got)
	}
}

func TestWorkerReserve(t *testing.T) {
	tests := []struct {
		name           string
		workerCapacity Capacity
		request        job.ResourceRequest
		wantErr        error
	}{
		{
			name:           "valid reserve",
			workerCapacity: Capacity{CPUCores: 4, MemoryMiB: 8192},
			request:        job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096},
			wantErr:        nil,
		},
		{
			name:           "exact fit reserve",
			workerCapacity: Capacity{CPUCores: 4, MemoryMiB: 8192},
			request:        job.ResourceRequest{CPUCores: 4, MemoryMiB: 8192},
			wantErr:        nil,
		},
		{
			name:           "insufficient CPU",
			workerCapacity: Capacity{CPUCores: 4, MemoryMiB: 8192},
			request:        job.ResourceRequest{CPUCores: 5, MemoryMiB: 4096},
			wantErr:        ErrInsufficientCapacity,
		},
		{
			name:           "insufficient memory",
			workerCapacity: Capacity{CPUCores: 4, MemoryMiB: 8192},
			request:        job.ResourceRequest{CPUCores: 2, MemoryMiB: 16384},
			wantErr:        ErrInsufficientCapacity,
		},
		{
			name:           "invalid request with zero CPU",
			workerCapacity: Capacity{CPUCores: 4, MemoryMiB: 8192},
			request:        job.ResourceRequest{CPUCores: 0, MemoryMiB: 4096},
			wantErr:        ErrInvalidResourceRequest,
		},
		{
			name:           "invalid request with zero memory",
			workerCapacity: Capacity{CPUCores: 4, MemoryMiB: 8192},
			request:        job.ResourceRequest{CPUCores: 2, MemoryMiB: 0},
			wantErr:        ErrInvalidResourceRequest,
		},
		{
			name:           "both resources insufficient",
			workerCapacity: Capacity{CPUCores: 4, MemoryMiB: 8192},
			request:        job.ResourceRequest{CPUCores: 5, MemoryMiB: 8193},
			wantErr:        ErrInsufficientCapacity,
		},
		{
			name:           "negative CPU",
			workerCapacity: Capacity{CPUCores: 4, MemoryMiB: 8192},
			request:        job.ResourceRequest{CPUCores: -1, MemoryMiB: 1024},
			wantErr:        ErrInvalidResourceRequest,
		},
		{
			name:           "negative memory",
			workerCapacity: Capacity{CPUCores: 4, MemoryMiB: 8192},
			request:        job.ResourceRequest{CPUCores: 1, MemoryMiB: -1},
			wantErr:        ErrInvalidResourceRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created, err := New("worker1", tt.workerCapacity)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			before := created.AvailableCapacity()
			err = created.Reserve(tt.request)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Reserve() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				if got := created.AvailableCapacity(); got != before {
					t.Errorf("AvailableCapacity() = %v, want unchanged %v", got, before)
				}
				return
			}

			expectedAvailable := Capacity{
				CPUCores:  tt.workerCapacity.CPUCores - tt.request.CPUCores,
				MemoryMiB: tt.workerCapacity.MemoryMiB - tt.request.MemoryMiB,
			}

			if created.AvailableCapacity() != expectedAvailable {
				t.Errorf("Available capacity = %v, want %v", created.AvailableCapacity(), expectedAvailable)
			}
		})
	}
}

func TestWorkerRelease(t *testing.T) {
	tests := []struct {
		name           string
		workerCapacity Capacity
		reserve        job.ResourceRequest
		release        job.ResourceRequest
		wantErr        error
	}{
		{
			name:           "valid release",
			workerCapacity: Capacity{CPUCores: 4, MemoryMiB: 8192},
			reserve:        job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096},
			release:        job.ResourceRequest{CPUCores: 1, MemoryMiB: 2048},
			wantErr:        nil,
		},
		{
			name:           "release all reserved",
			workerCapacity: Capacity{CPUCores: 4, MemoryMiB: 8192},
			reserve:        job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096},
			release:        job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096},
			wantErr:        nil,
		},
		{
			name:           "release exceeds usage",
			workerCapacity: Capacity{CPUCores: 4, MemoryMiB: 8192},
			reserve:        job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096},
			release:        job.ResourceRequest{CPUCores: 3, MemoryMiB: 4096},
			wantErr:        ErrReleaseExceedsUsage,
		},
		{
			name:           "release with zero CPU",
			workerCapacity: Capacity{CPUCores: 4, MemoryMiB: 8192},
			reserve:        job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096},
			release:        job.ResourceRequest{CPUCores: 0, MemoryMiB: 2048},
			wantErr:        ErrInvalidResourceRequest,
		},
		{
			name:           "release with zero memory",
			workerCapacity: Capacity{CPUCores: 4, MemoryMiB: 8192},
			reserve:        job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096},
			release:        job.ResourceRequest{CPUCores: 1, MemoryMiB: 0},
			wantErr:        ErrInvalidResourceRequest,
		},
		{
			name:           "release exceeds memory usage",
			workerCapacity: Capacity{CPUCores: 4, MemoryMiB: 8192},
			reserve:        job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096},
			release:        job.ResourceRequest{CPUCores: 2, MemoryMiB: 4097},
			wantErr:        ErrReleaseExceedsUsage,
		},
		{
			name:           "release with negative CPU",
			workerCapacity: Capacity{CPUCores: 4, MemoryMiB: 8192},
			reserve:        job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096},
			release:        job.ResourceRequest{CPUCores: -1, MemoryMiB: 1024},
			wantErr:        ErrInvalidResourceRequest,
		},
		{
			name:           "release with negative memory",
			workerCapacity: Capacity{CPUCores: 4, MemoryMiB: 8192},
			reserve:        job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096},
			release:        job.ResourceRequest{CPUCores: 1, MemoryMiB: -1},
			wantErr:        ErrInvalidResourceRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created, err := New("worker1", tt.workerCapacity)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			if err := created.Reserve(tt.reserve); err != nil {
				t.Fatalf("Reserve() setup error = %v", err)
			}

			before := created.AvailableCapacity()
			err = created.Release(tt.release)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Release() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				if got := created.AvailableCapacity(); got != before {
					t.Errorf("AvailableCapacity() = %v, want unchanged %v", got, before)
				}
				return
			}

			expectedAvailable := Capacity{
				CPUCores:  before.CPUCores + tt.release.CPUCores,
				MemoryMiB: before.MemoryMiB + tt.release.MemoryMiB,
			}

			if created.AvailableCapacity() != expectedAvailable {
				t.Errorf("Available capacity = %v, want %v", created.AvailableCapacity(), expectedAvailable)
			}
		})
	}
}

func TestWorkerReleaseWithoutReservation(t *testing.T) {
	w, err := New("worker1", Capacity{CPUCores: 4, MemoryMiB: 8192})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	before := w.AvailableCapacity()

	err = w.Release(job.ResourceRequest{
		CPUCores:  1,
		MemoryMiB: 2048,
	})

	if !errors.Is(err, ErrReleaseExceedsUsage) {
		t.Errorf("Release() error = %v, want %v", err, ErrReleaseExceedsUsage)
	}

	if got := w.AvailableCapacity(); got != before {
		t.Errorf("AvailableCapacity() = %v, want unchanged %v", got, before)
	}
}

func TestCanRunUsesAvailableCapacity(t *testing.T) {
	w, err := New(
		"worker1",
		Capacity{
			CPUCores:  4,
			MemoryMiB: 8192,
		},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	request := job.ResourceRequest{
		CPUCores:  3,
		MemoryMiB: 5000,
	}

	reservation := job.ResourceRequest{
		CPUCores:  2,
		MemoryMiB: 4096,
	}

	// Initially: 3/5000 fits within 4/8192.
	if !w.CanRun(request) {
		t.Fatal("CanRun() = false before reservation, want true")
	}

	if err := w.Reserve(reservation); err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}

	// After reservation, only 2/4096 remains.
	if w.CanRun(request) {
		t.Fatal("CanRun() = true after reservation, want false")
	}

	if err := w.Release(reservation); err != nil {
		t.Fatalf("Release() error = %v", err)
	}

	// Releasing restores capacity to 4/8192.
	if !w.CanRun(request) {
		t.Fatal("CanRun() = false after release, want true")
	}
}

func TestWorkerReserveSequentially(t *testing.T) {
	w, err := New(
		"worker1",
		Capacity{
			CPUCores:  4,
			MemoryMiB: 8192,
		},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	request := job.ResourceRequest{
		CPUCores:  2,
		MemoryMiB: 4096,
	}

	if err := w.Reserve(request); err != nil {
		t.Fatalf("first Reserve() error = %v", err)
	}

	if err := w.Reserve(request); err != nil {
		t.Fatalf("second Reserve() error = %v", err)
	}

	wantEmpty := Capacity{}

	if got := w.AvailableCapacity(); got != wantEmpty {
		t.Fatalf("AvailableCapacity() = %v, want %v", got, wantEmpty)
	}

	before := w.AvailableCapacity()

	err = w.Reserve(request)
	if !errors.Is(err, ErrInsufficientCapacity) {
		t.Errorf("third Reserve() error = %v, want %v",
			err, ErrInsufficientCapacity)
	}

	if got := w.AvailableCapacity(); got != before {
		t.Errorf("AvailableCapacity() = %v, want unchanged %v",
			got, before)
	}
}

func TestConcurrencyWrite(t *testing.T) {
	w, err := New("worker1", Capacity{CPUCores: 2, MemoryMiB: 1024})

	if err != nil {
		t.Fatalf("New() returns error - %v", err)
	}

	results := make([]error, 2)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		results[0] = w.Reserve(job.ResourceRequest{CPUCores: 2, MemoryMiB: 1024})
	}()

	go func() {
		defer wg.Done()

		results[1] = w.Reserve(job.ResourceRequest{CPUCores: 2, MemoryMiB: 1024})
	}()

	wg.Wait()

	successes := 0
	for _, err := range results {
		if err == nil {
			successes++
		}
	}

	if successes != 1 {
		t.Fatalf("successful reservations = %d, want 1; results = %v", successes, results)
	}

	if (results[0] == nil) && (results[1] == nil) {
		t.Fatalf("expected exactly one successful reservation: %v", results)
	}

	if results[0] != nil && results[0] != ErrInsufficientCapacity {
		t.Fatalf("unexpected first result: %v", results[0])
	}

	if results[1] != nil && results[1] != ErrInsufficientCapacity {
		t.Fatalf("unexpected second result: %v", results[1])
	}

	if w.AvailableCapacity().CPUCores != 0 || w.AvailableCapacity().MemoryMiB != 0 {
		t.Fatalf("Reserve() failed")
	}
}

func TestConcurrencyReadWrite(t *testing.T) {
	w, err := New("worker1", Capacity{CPUCores: 2, MemoryMiB: 1024})

	if err != nil {
		t.Fatalf("New() returns error - %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		for i := 0; i < 20; i++ {
			err := w.Reserve(job.ResourceRequest{CPUCores: 2, MemoryMiB: 1024})

			if err != nil {
				t.Errorf("Reserve() returns error - %v", err)
			}

			err = w.Release(job.ResourceRequest{CPUCores: 2, MemoryMiB: 1024})

			if err != nil {
				t.Errorf("Release() returns error - %v", err)
			}
		}
	}()

	go func() {
		defer wg.Done()

		for i := 0; i < 20; i++ {
			_ = w.AvailableCapacity()
			_ = w.CanRun(job.ResourceRequest{
				CPUCores:  2,
				MemoryMiB: 1024,
			})
		}
	}()

	wg.Wait()

	if w.AvailableCapacity().CPUCores != 2 || w.AvailableCapacity().MemoryMiB != 1024 {
		t.Fatalf("Restoring capacity fails")
	}
}
