package agent

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/sanketlengare/orchard/internal/client"
	"github.com/sanketlengare/orchard/internal/executor"
)

type fakeClient struct {
	assignment client.Assignment
	job        client.Job

	finishedJobID  string
	finishedStatus string
	finishCalls    int
	finishErr      error
}

func (f *fakeClient) GetAssignment(ctx context.Context, id string) (client.Assignment, error) {
	return f.assignment, nil
}

func (f *fakeClient) GetJob(ctx context.Context, id string) (client.Job, error) {
	return f.job, nil
}

func (f *fakeClient) FinishJob(ctx context.Context, id string, status string) error {
	f.finishedJobID = id
	f.finishedStatus = status
	f.finishCalls++
	return f.finishErr
}

func TestRunJobSuccess(t *testing.T) {
	wantCommand := []string{"echo", "hello"}
	api := &fakeClient{
		assignment: client.Assignment{
			JobID:    "job1",
			WorkerID: "worker1",
		},
		job: client.Job{
			ID:      "job1",
			Command: wantCommand,
			Status:  "running",
		},
	}

	wantResult := executor.Result{
		Stdout:   "hello\n",
		ExitCode: 0,
	}
	var gotCommand []string
	executeCalls := 0
	execute := func(ctx context.Context, command []string) (executor.Result, error) {
		gotCommand = slices.Clone(command)
		executeCalls++
		return wantResult, nil
	}

	runner := New("worker1", api, execute)
	result, err := runner.RunJob(t.Context(), "job1")
	if err != nil {
		t.Fatalf("RunJob() error = %v", err)
	}

	if executeCalls != 1 {
		t.Errorf("execution calls = %d, want 1", executeCalls)
	}
	if !slices.Equal(gotCommand, wantCommand) {
		t.Errorf("executed command = %v, want %v", gotCommand, wantCommand)
	}
	if result != wantResult {
		t.Errorf("result = %+v, want %+v", result, wantResult)
	}
	if api.finishCalls != 1 {
		t.Errorf("completion calls = %d, want 1", api.finishCalls)
	}
	if api.finishedJobID != "job1" {
		t.Errorf("finished job ID = %q, want %q", api.finishedJobID, "job1")
	}
	if api.finishedStatus != "succeeded" {
		t.Errorf("finished status = %q, want %q", api.finishedStatus, "succeeded")
	}
}

func TestRunJobExecutionFailure(t *testing.T) {
	api := &fakeClient{
		assignment: client.Assignment{
			JobID:    "job1",
			WorkerID: "worker1",
		},
		job: client.Job{
			ID:      "job1",
			Command: []string{"echo", "hello"},
			Status:  "running",
		},
	}

	executionErr := errors.New("command failed")

	wantResult := executor.Result{
		Stdout:   "partial output\n",
		Stderr:   "something went wrong\n",
		ExitCode: 7,
	}
	executeCalls := 0
	execute := func(ctx context.Context, command []string) (executor.Result, error) {
		executeCalls++
		return wantResult, executionErr
	}

	runner := New("worker1", api, execute)
	result, err := runner.RunJob(t.Context(), "job1")
	if !errors.Is(err, executionErr) {
		t.Fatalf("RunJob() error = %v, want %v", err, executionErr)
	}
	if result != wantResult {
		t.Errorf("result = %+v, want %+v", result, wantResult)
	}
	if executeCalls != 1 {
		t.Errorf("execution calls = %d, want 1", executeCalls)
	}
	if api.finishCalls != 1 {
		t.Errorf("completion calls = %d, want 1", api.finishCalls)
	}
	if api.finishedJobID != "job1" {
		t.Errorf("finished job ID = %q, want %q", api.finishedJobID, "job1")
	}
	if api.finishedStatus != "failed" {
		t.Errorf("finished status = %q, want %q", api.finishedStatus, "failed")
	}
}

func TestRunJobRejectsJob(t *testing.T) {
	tests := []struct {
		name     string
		workerID string
		status   string
		wantErr  error
	}{
		{
			name:     "wrong worker",
			workerID: "worker2",
			status:   "running",
			wantErr:  ErrAssignmentMismatch,
		},
		{
			name:     "job not running",
			workerID: "worker1",
			status:   "pending",
			wantErr:  ErrJobNotRunning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &fakeClient{
				assignment: client.Assignment{JobID: "job1", WorkerID: tt.workerID},
				job: client.Job{
					ID:      "job1",
					Command: []string{"echo", "hello"},
					Status:  tt.status,
				},
			}
			executeCalls := 0
			execute := func(ctx context.Context, command []string) (executor.Result, error) {
				executeCalls++
				return executor.Result{}, nil
			}

			result, err := New("worker1", api, execute).RunJob(t.Context(), "job1")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("RunJob() error = %v, want %v", err, tt.wantErr)
			}
			if result != (executor.Result{ExitCode: -1}) {
				t.Errorf("result = %+v, want no execution with exit code -1", result)
			}
			if executeCalls != 0 {
				t.Errorf("execution calls = %d, want 0", executeCalls)
			}
			if api.finishCalls != 0 {
				t.Errorf("completion calls = %d, want 0", api.finishCalls)
			}
		})
	}
}

func TestRunJobReportingFailure(t *testing.T) {
	executionErr := errors.New("command failed")
	reportingErr := errors.New("completion request failed")
	tests := []struct {
		name       string
		result     executor.Result
		executeErr error
		wantStatus string
	}{
		{
			name:       "successful execution",
			result:     executor.Result{Stdout: "hello\n", ExitCode: 0},
			wantStatus: "succeeded",
		},
		{
			name:       "failed execution",
			result:     executor.Result{Stderr: "failure\n", ExitCode: 7},
			executeErr: executionErr,
			wantStatus: "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &fakeClient{
				assignment: client.Assignment{JobID: "job1", WorkerID: "worker1"},
				job: client.Job{
					ID:      "job1",
					Command: []string{"echo", "hello"},
					Status:  "running",
				},
				finishErr: reportingErr,
			}
			executeCalls := 0
			execute := func(ctx context.Context, command []string) (executor.Result, error) {
				executeCalls++
				return tt.result, tt.executeErr
			}

			result, err := New("worker1", api, execute).RunJob(t.Context(), "job1")
			if !errors.Is(err, reportingErr) {
				t.Fatalf("RunJob() error = %v, want reporting error %v", err, reportingErr)
			}
			if tt.executeErr != nil && !errors.Is(err, tt.executeErr) {
				t.Errorf("RunJob() error = %v, want execution error %v preserved", err, tt.executeErr)
			}
			if result != tt.result {
				t.Errorf("result = %+v, want %+v", result, tt.result)
			}
			if executeCalls != 1 {
				t.Errorf("execution calls = %d, want 1", executeCalls)
			}
			if api.finishCalls != 1 {
				t.Errorf("completion calls = %d, want 1", api.finishCalls)
			}
			if api.finishedJobID != "job1" || api.finishedStatus != tt.wantStatus {
				t.Errorf("completion = (%q, %q), want (%q, %q)",
					api.finishedJobID, api.finishedStatus, "job1", tt.wantStatus)
			}
		})
	}
}

func TestRunJobNonzeroExitWithoutError(t *testing.T) {
	api := &fakeClient{
		assignment: client.Assignment{JobID: "job1", WorkerID: "worker1"},
		job: client.Job{
			ID:      "job1",
			Command: []string{"echo", "hello"},
			Status:  "running",
		},
	}
	wantResult := executor.Result{Stdout: "partial output\n", ExitCode: 7}
	execute := func(ctx context.Context, command []string) (executor.Result, error) {
		return wantResult, nil
	}

	result, err := New("worker1", api, execute).RunJob(t.Context(), "job1")
	if err == nil {
		t.Fatal("RunJob() error = nil, want nonzero exit error")
	}
	if !strings.Contains(err.Error(), "7") {
		t.Errorf("error = %q, want exit code 7", err.Error())
	}
	if result != wantResult {
		t.Errorf("result = %+v, want %+v", result, wantResult)
	}
	if api.finishCalls != 1 || api.finishedJobID != "job1" || api.finishedStatus != "failed" {
		t.Errorf("completion = (%d calls, %q, %q), want (1 call, job1, failed)",
			api.finishCalls, api.finishedJobID, api.finishedStatus)
	}
}
