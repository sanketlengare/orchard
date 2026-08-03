package job

import (
	"testing"
)

func TestStatusCanTransitionTo(t *testing.T) {
	tests := []struct {
		name     string
		current  Status
		next     Status
		expected bool
	}{
		{
			name:     "pending to running",
			current:  StatusPending,
			next:     StatusRunning,
			expected: true,
		},
		{
			name:     "running to succeeded",
			current:  StatusRunning,
			next:     StatusSucceeded,
			expected: true,
		},
		{
			name:     "running to failed",
			current:  StatusRunning,
			next:     StatusFailed,
			expected: true,
		},
		{
			name:     "pending to succeeded",
			current:  StatusPending,
			next:     StatusSucceeded,
			expected: false,
		},
		{
			name:     "succeeded to succeeded",
			current:  StatusSucceeded,
			next:     StatusSucceeded,
			expected: false,
		},

		{
			name:     "failed to unknown",
			current:  StatusFailed,
			next:     Status("unknown"),
			expected: false,
		},
		{
			name:     "pending to unknown",
			current:  StatusPending,
			next:     Status("unknown"),
			expected: false,
		},
		{
			name:     "pending to pending",
			current:  StatusPending,
			next:     StatusPending,
			expected: false,
		},
		{
			name:     "running to pending",
			current:  StatusRunning,
			next:     StatusPending,
			expected: false,
		},
		{
			name:     "running to running",
			current:  StatusRunning,
			next:     StatusRunning,
			expected: false,
		},
		{
			name:     "succeeded to running",
			current:  StatusSucceeded,
			next:     StatusRunning,
			expected: false,
		},
		{
			name:     "failed to running",
			current:  StatusFailed,
			next:     StatusRunning,
			expected: false,
		},
		{
			name:     "unknown to running",
			current:  Status("paused"),
			next:     StatusRunning,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.current.CanTransitionTo(tt.next)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestStatusValues(t *testing.T) {
	tests := []struct {
		name string
		got  Status
		want Status
	}{
		{name: "pending", got: StatusPending, want: Status("pending")},
		{name: "running", got: StatusRunning, want: Status("running")},
		{name: "succeeded", got: StatusSucceeded, want: Status("succeeded")},
		{name: "failed", got: StatusFailed, want: Status("failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("status = %q, want %q", tt.got, tt.want)
			}
		})
	}
}
