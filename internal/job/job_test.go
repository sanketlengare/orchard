package job

import (
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
)

func validResourceRequest() ResourceRequest {
	return ResourceRequest{
		CPUCores:  1,
		MemoryMiB: 128,
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		command []string
		wantErr error
	}{
		{
			name:    "valid job",
			id:      "job1",
			command: []string{"echo", "hello"},
			wantErr: nil,
		},
		{
			name:    "missing id",
			id:      "",
			command: []string{"echo", "hello"},
			wantErr: ErrIDRequired,
		},
		{
			name:    "missing command",
			id:      "job2",
			command: nil,
			wantErr: ErrCommandRequired,
		},
		{
			name:    "empty command",
			id:      "job3",
			command: []string{},
			wantErr: ErrCommandRequired,
		},
		{
			name:    "command with empty first element",
			id:      "job4",
			command: []string{""},
			wantErr: ErrCommandRequired,
		},
		{
			name:    "command with empty first element and other elements",
			id:      "job5",
			command: []string{"", "arg1"},
			wantErr: ErrCommandRequired,
		},
		{
			name:    "valid job with multiple arguments",
			id:      "job6",
			command: []string{"echo", "hello", "world"},
			wantErr: nil,
		},
		{
			name:    "valid job with first arg but second arg empty",
			id:      "job7",
			command: []string{"echo", ""},
			wantErr: nil,
		},
		{
			name:    "empty space id",
			id:      "   ",
			command: []string{"echo", "hello"},
			wantErr: ErrIDRequired,
		},
		{
			name:    "id with leading and trailing spaces",
			id:      "  job8  ",
			command: []string{"echo", "hello"},
			wantErr: nil,
		},
		{
			name:    "blank executable",
			id:      "job9",
			command: []string{"   "},
			wantErr: ErrCommandRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j, err := New(tt.id, tt.command, validResourceRequest())
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if j.ID() != strings.TrimSpace(tt.id) {
					t.Errorf("New() id = %v, want %v", j.ID(), strings.TrimSpace(tt.id))
				}
				if len(j.Command()) != len(tt.command) {
					t.Errorf("New() command length = %v, want %v", len(j.Command()), len(tt.command))
				} else {
					for i := range j.Command() {
						if j.Command()[i] != tt.command[i] {
							t.Errorf("New() command[%d] = %v, want %v", i, j.Command()[i], tt.command[i])
						}
					}
				}
				if j.Status() != StatusPending {
					t.Errorf("New() status = %v, want %v", j.Status(), StatusPending)
				}
			}

			if tt.wantErr != nil && j != nil {
				t.Errorf("New() job = %v, want nil", j)
			}
		})
	}
}

func TestNewCopiesCommand(t *testing.T) {
	originalCommand := []string{"echo", "hello"}
	j, err := New("job1", originalCommand, validResourceRequest())
	if err != nil {
		t.Fatalf("New() error = %v, wantErr %v", err, nil)
	}

	// Modify the original command slice
	originalCommand[0] = "modified"

	want := []string{"echo", "hello"}

	if got := j.Command(); !slices.Equal(got, want) {
		t.Errorf("Command() = %v, want %v", got, want)
	}
}

func TestJobCommandReturnsCopy(t *testing.T) {
	j, err := New("job1", []string{"echo", "hello"}, validResourceRequest())
	if err != nil {
		t.Fatalf("New() error = %v, wantErr %v", err, nil)
	}

	cmdCopy := j.Command()
	cmdCopy[0] = "modified"

	want := []string{"echo", "hello"}

	if got := j.Command(); !slices.Equal(got, want) {
		t.Errorf("Command() = %v, want %v", got, want)
	}
}

func TestTransitionTo(t *testing.T) {
	tests := []struct {
		name       string
		current    Status
		next       Status
		wantStatus Status
		wantErr    error
	}{
		{
			name:       "pending to running",
			current:    StatusPending,
			next:       StatusRunning,
			wantStatus: StatusRunning,
			wantErr:    nil,
		},
		{
			name:       "running to succeeded",
			current:    StatusRunning,
			next:       StatusSucceeded,
			wantStatus: StatusSucceeded,
			wantErr:    nil,
		},
		{
			name:       "running to failed",
			current:    StatusRunning,
			next:       StatusFailed,
			wantStatus: StatusFailed,
			wantErr:    nil,
		},
		{
			name:       "pending to succeeded",
			current:    StatusPending,
			next:       StatusSucceeded,
			wantStatus: StatusPending,
			wantErr:    ErrInvalidTransition,
		},
		{
			name:       "succeeded to running",
			current:    StatusSucceeded,
			next:       StatusRunning,
			wantStatus: StatusSucceeded,
			wantErr:    ErrInvalidTransition,
		},
		{
			name:       "failed to running",
			current:    StatusFailed,
			next:       StatusRunning,
			wantStatus: StatusFailed,
			wantErr:    ErrInvalidTransition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &Job{
				id:      "job1",
				command: []string{"echo", "hello"},
				status:  tt.current,
			}

			err := j.TransitionTo(tt.next)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("TransitionTo() error = %v, want %v", err, tt.wantErr)
			}
			if got := j.Status(); got != tt.wantStatus {
				t.Errorf("Status() = %q, want %q", got, tt.wantStatus)
			}
		})
	}
}

func TestNewValidatesResourceRequest(t *testing.T) {
	tests := []struct {
		name      string
		resources ResourceRequest
		wantErr   error
	}{
		{
			name: "valid resources",
			resources: ResourceRequest{
				CPUCores:  1,
				MemoryMiB: 128,
			},
			wantErr: nil,
		},
		{
			name: "large valid resources",
			resources: ResourceRequest{
				CPUCores:  16,
				MemoryMiB: 32768,
			},
			wantErr: nil,
		},
		{
			name: "zero CPU cores",
			resources: ResourceRequest{
				CPUCores:  0,
				MemoryMiB: 128,
			},
			wantErr: ErrCPUCoresRequired,
		},
		{
			name: "negative CPU cores",
			resources: ResourceRequest{
				CPUCores:  -1,
				MemoryMiB: 128,
			},
			wantErr: ErrCPUCoresRequired,
		},
		{
			name: "zero memory",
			resources: ResourceRequest{
				CPUCores:  1,
				MemoryMiB: 0,
			},
			wantErr: ErrMemoryMiBRequired,
		},
		{
			name: "negative memory",
			resources: ResourceRequest{
				CPUCores:  1,
				MemoryMiB: -128,
			},
			wantErr: ErrMemoryMiBRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created, err := New(
				"job1",
				[]string{"echo", "hello"},
				tt.resources,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("New() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				if created != nil {
					t.Errorf("New() job = %v, want nil", created)
				}
				return
			}

			if created == nil {
				t.Fatal("New() job = nil, want a job")
			}

			if got := created.Resources(); got != tt.resources {
				t.Errorf("Resources() = %v, want %v", got, tt.resources)
			}
		})
	}
}

func TestResourceRequestReturnsCopy(t *testing.T) {
	j, err := New("job1", []string{"echo", "hello"}, validResourceRequest())
	if err != nil {
		t.Fatalf("New() error = %v, wantErr %v", err, nil)
	}

	resourcesCopy := j.Resources()
	resourcesCopy.CPUCores = 2
	resourcesCopy.MemoryMiB = 256

	want := ResourceRequest{
		CPUCores:  1,
		MemoryMiB: 128,
	}

	if got := j.Resources(); got != want {
		t.Errorf("Resources() = %v, want %v", got, want)
	}
}

func TestConcurrency(t *testing.T) {
	j, err := New("job1", []string{"echo", "hello"}, validResourceRequest())

	if err != nil {
		t.Fatalf("New() error = %v, wantErr %v", err, nil)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		for i := 0; i < 20; i++ {
			j.Status()
		}
	}()

	go func() {
		defer wg.Done()

		err := j.TransitionTo(StatusRunning)

		if err != nil {
			t.Errorf("TransitionTo() returns error - %v", err)
		}

		err = j.TransitionTo(StatusSucceeded)

		if err != nil {
			t.Errorf("TransitionTo() returns error - %v", err)
		}
	}()

	wg.Wait()

	if j.Status() != StatusSucceeded {
		t.Fatalf("Want status - %v, got - %v", StatusSucceeded, j.Status())
	}

}
