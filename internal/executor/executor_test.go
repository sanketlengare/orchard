package executor

import (
	"context"
	"errors"
	"testing"
)

func TestRunSuccess(t *testing.T) {
	result, err := Run(context.Background(), []string{"echo", "hello orchard"})

	if err != nil {
		t.Fatalf("Run() returns error - %v", err)
	}

	if result.ExitCode != 0 {
		t.Fatalf("Exit code expected - %v, but got - %v", 0, result.ExitCode)
	}

	if result.Stdout != "hello orchard\n" {
		t.Fatalf("std output expected - %v, but got - %v", "hello orchard", result.Stdout)
	}

	if result.Stderr != "" {
		t.Fatalf("stderr expected to be empty, but got %q", result.Stderr)
	}
}

func TestRunFailure(t *testing.T) {
	result, err := Run(context.Background(), []string{"/bin/sh", "-c", "printf 'failure' >&2; exit 7"})

	if err == nil {
		t.Fatalf("Run() should return error")
	}

	if result.ExitCode != 7 {
		t.Fatalf("Exit code expected - %v, but got - %v", 7, result.ExitCode)
	}

	if result.Stdout != "" {
		t.Fatalf("std output expected to be empty, but got - %v", result.Stdout)
	}

	if result.Stderr != "failure" {
		t.Fatalf("stderr expected to be failure, but got %q", result.Stderr)
	}
}

func TestRunFailureCommandRequired(t *testing.T) {
	tests := []struct {
		name    string
		command []string
	}{
		{name: "no content in list", command: []string{}},
		{name: "empty string", command: []string{""}},
		{name: "empty string with space", command: []string{"	"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Run(context.Background(), tt.command)

			if !errors.Is(err, ErrCommandRequired) {
				t.Fatalf("Run() expected error - %v, but got %v", ErrCommandRequired, err)
			}

			if result.ExitCode != -1 {
				t.Fatalf("Exit code expected - %v, but got - %v", -1, result.ExitCode)
			}

			if result.Stdout != "" {
				t.Fatalf("std output expected to be empty, but got - %v", result.Stdout)
			}

			if result.Stderr != "" {
				t.Fatalf("stderr expected to be empty, but got %q", result.Stderr)
			}
		})
	}
}
