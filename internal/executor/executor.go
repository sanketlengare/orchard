package executor

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
)

var ErrCommandRequired = errors.New("executor: command required")

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

func Run(ctx context.Context, command []string) (Result, error) {
	if len(command) == 0 || strings.TrimSpace(command[0]) == "" {
		return Result{ExitCode: -1}, ErrCommandRequired
	}

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)

	var stdOutBuffer bytes.Buffer
	var stdErrBuffer bytes.Buffer

	cmd.Stdout = &stdOutBuffer
	cmd.Stderr = &stdErrBuffer

	err := cmd.Run()
	result := Result{
		Stdout:   stdOutBuffer.String(),
		Stderr:   stdErrBuffer.String(),
		ExitCode: 0,
	}

	if err != nil {
		result.ExitCode = -1

		if cmd.ProcessState != nil {
			result.ExitCode = cmd.ProcessState.ExitCode()
		}

		return result, err
	}

	return result, nil
}
