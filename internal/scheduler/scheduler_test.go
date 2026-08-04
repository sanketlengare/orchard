package scheduler

import (
	"errors"
	"testing"

	"github.com/sanketlengare/orchard/internal/job"
	"github.com/sanketlengare/orchard/internal/worker"
)

func mustWorker(t *testing.T, id string, total worker.Capacity) *worker.Worker {
	t.Helper()
	w, err := worker.New(id, total)
	if err != nil {
		t.Fatalf("failed to create worker: %v", err)
	}
	return w
}

func TestSelect(t *testing.T) {
	tests := []struct {
		name      string
		workers   []*worker.Worker
		request   job.ResourceRequest
		wantIndex int
		wantErr   error
	}{
		{
			name:      "first worker can run request",
			workers:   []*worker.Worker{mustWorker(t, "worker1", worker.Capacity{CPUCores: 4, MemoryMiB: 8192})},
			request:   job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096},
			wantIndex: 0,
			wantErr:   nil,
		},
		{
			name: "First cannot run it, but second can",
			workers: []*worker.Worker{
				mustWorker(t, "worker1", worker.Capacity{CPUCores: 1, MemoryMiB: 1024}),
				mustWorker(t, "worker2", worker.Capacity{CPUCores: 4, MemoryMiB: 8192}),
			},
			request:   job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096},
			wantIndex: 1,
			wantErr:   nil,
		},
		{
			name: "Multiple workers can run it; first compatible worker wins",
			workers: []*worker.Worker{
				mustWorker(t, "worker1", worker.Capacity{CPUCores: 4, MemoryMiB: 8192}),
				mustWorker(t, "worker2", worker.Capacity{CPUCores: 4, MemoryMiB: 8192}),
			},
			request:   job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096},
			wantIndex: 0,
			wantErr:   nil,
		},
		{
			name: "Exact-capacity fit works",
			workers: []*worker.Worker{
				mustWorker(t, "worker1", worker.Capacity{CPUCores: 2, MemoryMiB: 4096}),
			},
			request:   job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096},
			wantIndex: 0,
			wantErr:   nil,
		},
		{
			name: "No worker has enough CPU",
			workers: []*worker.Worker{
				mustWorker(t, "worker1", worker.Capacity{CPUCores: 1, MemoryMiB: 8192}),
				mustWorker(t, "worker2", worker.Capacity{CPUCores: 1, MemoryMiB: 8192}),
			},
			request:   job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096},
			wantIndex: -1,
			wantErr:   ErrNoSuitableWorker,
		},
		{
			name: "No worker has enough memory",
			workers: []*worker.Worker{
				mustWorker(t, "worker1", worker.Capacity{CPUCores: 4, MemoryMiB: 2048}),
				mustWorker(t, "worker2", worker.Capacity{CPUCores: 4, MemoryMiB: 2048}),
			},
			request:   job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096},
			wantIndex: -1,
			wantErr:   ErrNoSuitableWorker,
		},
		{
			name:      "Empty worker slice",
			workers:   []*worker.Worker{},
			request:   job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096},
			wantIndex: -1,
			wantErr:   ErrNoSuitableWorker,
		},
		{
			name:      "Nil worker slice",
			workers:   nil,
			request:   job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096},
			wantIndex: -1,
			wantErr:   ErrNoSuitableWorker,
		},
		{
			name: "Nil worker in slice",
			workers: []*worker.Worker{
				nil,
				mustWorker(t, "worker1", worker.Capacity{CPUCores: 4, MemoryMiB: 8192}),
			},
			request:   job.ResourceRequest{CPUCores: 2, MemoryMiB: 4096},
			wantIndex: 1,
			wantErr:   nil,
		},
		{
			name: "Zero CPU request",
			workers: []*worker.Worker{
				mustWorker(t, "worker1", worker.Capacity{CPUCores: 4, MemoryMiB: 8192}),
			},
			request:   job.ResourceRequest{CPUCores: 0, MemoryMiB: 4096},
			wantIndex: -1,
			wantErr:   ErrInvalidResourceRequest,
		},
		{
			name: "Zero Memory request",
			workers: []*worker.Worker{
				mustWorker(t, "worker1", worker.Capacity{CPUCores: 4, MemoryMiB: 8192}),
			},
			request:   job.ResourceRequest{CPUCores: 2, MemoryMiB: 0},
			wantIndex: -1,
			wantErr:   ErrInvalidResourceRequest,
		},
		{
			name: "Negative CPU request",
			workers: []*worker.Worker{
				mustWorker(t, "worker1", worker.Capacity{CPUCores: 4, MemoryMiB: 8192}),
			},
			request:   job.ResourceRequest{CPUCores: -1, MemoryMiB: 4096},
			wantIndex: -1,
			wantErr:   ErrInvalidResourceRequest,
		},
		{
			name: "Negative Memory request",
			workers: []*worker.Worker{
				mustWorker(t, "worker1", worker.Capacity{CPUCores: 4, MemoryMiB: 8192}),
			},
			request:   job.ResourceRequest{CPUCores: 2, MemoryMiB: -1024},
			wantIndex: -1,
			wantErr:   ErrInvalidResourceRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selected, err := Select(tt.workers, tt.request)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Select() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantIndex == -1 {
				if selected != nil {
					t.Fatalf("Select() worker = %v, want nil", selected)
				}
				return
			}

			want := tt.workers[tt.wantIndex]
			if selected != want {
				t.Fatalf("Select() worker = %v, want %v", selected, want)
			}
		})
	}
}

func TestSelectUsesAvailableCapacity(t *testing.T) {
	first := mustWorker(t, "worker1", worker.Capacity{
		CPUCores:  4,
		MemoryMiB: 8192,
	})
	second := mustWorker(t, "worker2", worker.Capacity{
		CPUCores:  4,
		MemoryMiB: 8192,
	})

	request := job.ResourceRequest{
		CPUCores:  3,
		MemoryMiB: 4096,
	}

	if err := first.Reserve(request); err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}

	selected, err := Select([]*worker.Worker{first, second}, request)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}

	if selected != second {
		t.Fatalf("Select() worker = %v, want second worker", selected)
	}
}

func TestSelectDoesNotReserveCapacity(t *testing.T) {
	candidate := mustWorker(t, "worker1", worker.Capacity{
		CPUCores:  4,
		MemoryMiB: 8192,
	})
	request := job.ResourceRequest{
		CPUCores:  2,
		MemoryMiB: 4096,
	}
	before := candidate.AvailableCapacity()

	selected, err := Select([]*worker.Worker{candidate}, request)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if selected != candidate {
		t.Fatalf("Select() worker = %v, want %v", selected, candidate)
	}

	after := candidate.AvailableCapacity()
	if after != before {
		t.Fatalf("available capacity = %+v, want unchanged %+v", after, before)
	}
}
