package agent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sanketlengare/orchard/internal/client"
	"github.com/sanketlengare/orchard/internal/executor"
)

var ErrAssignmentMismatch = errors.New("agent: assignment mismatched")
var ErrJobMismatch = errors.New("agent: job mismatched")
var ErrJobNotRunning = errors.New("agent: job not running")

type ControlPlaneClient interface {
	GetAssignment(context.Context, string) (client.Assignment, error)
	GetJob(context.Context, string) (client.Job, error)
	FinishJob(context.Context, string, string) error
}

type ExecuteFunc func(
	context.Context,
	[]string,
) (executor.Result, error)

type Runner struct {
	workerID string
	client   ControlPlaneClient
	execute  ExecuteFunc
}

func New(workerID string, api ControlPlaneClient, execute ExecuteFunc) *Runner {
	return &Runner{
		workerID: workerID,
		client:   api,
		execute:  execute,
	}
}

func (r *Runner) RunJob(ctx context.Context, jobID string) (executor.Result, error) {

	assignment, err := r.client.GetAssignment(ctx, jobID)

	if err != nil {
		return executor.Result{ExitCode: -1}, err
	}

	if assignment.JobID != jobID || assignment.WorkerID != r.workerID {
		return executor.Result{ExitCode: -1}, ErrAssignmentMismatch
	}

	j, err := r.client.GetJob(ctx, jobID)

	if err != nil {
		return executor.Result{ExitCode: -1}, err
	}

	if j.ID != jobID {
		return executor.Result{ExitCode: -1}, ErrJobMismatch
	}

	if j.Status != "running" {
		return executor.Result{ExitCode: -1}, ErrJobNotRunning
	}

	result, executeErr := r.execute(ctx, j.Command)

	finalStatus := "succeeded"

	if executeErr != nil || result.ExitCode != 0 {
		finalStatus = "failed"

		if executeErr == nil {
			executeErr = fmt.Errorf(
				"command exited with code %d",
				result.ExitCode,
			)
		}
	}

	reportCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	reportingErr := r.client.FinishJob(reportCtx, jobID, finalStatus)

	return result, errors.Join(executeErr, reportingErr)
}
