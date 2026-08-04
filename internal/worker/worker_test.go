package worker

import (
	"errors"
	"strings"
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

	// Check if the original values in the worker remain unchanged
	if got := w.TotalCapacity(); got.CPUCores != 4 || got.MemoryMiB != 8192 {
		t.Errorf("TotalCapacity() = %v, want {CPUCores: 4, MemoryMiB: 8192}", got)
	}

	if got := w.AvailableCapacity(); got.CPUCores != 4 || got.MemoryMiB != 8192 {
		t.Errorf("AvailableCapacity() = %v, want {CPUCores: 4, MemoryMiB: 8192}", got)
	}
}
