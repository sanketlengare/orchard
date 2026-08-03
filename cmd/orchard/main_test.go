package main

import (
	"bytes"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"version"}, &stdout, &stderr)

	if code != 0 {
		t.Errorf("code = %d, want 0", code)
	}

	if got := stdout.String(); got != "orchard dev\n" {
		t.Errorf("stdout = %q, want %q", got, "orchard dev\n")
	}

	if got := stderr.String(); got != "" {
		t.Errorf("stderr = %q, want empty output", got)
	}
}

func TestRunMissingCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{}, &stdout, &stderr)

	if code != 2 {
		t.Errorf("code = %d, want 2", code)
	}

	if got := stdout.String(); got != "" {
		t.Errorf("stdout = %q, want empty output", got)
	}

	if got := stderr.String(); got != "usage: orchard <command>\n" {
		t.Errorf("stderr = %q, want %q", got, "usage: orchard <command>\n")
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"unknown"}, &stdout, &stderr)

	if code != 2 {
		t.Errorf("code = %d, want 2", code)
	}

	if got := stdout.String(); got != "" {
		t.Errorf("stdout = %q, want empty output", got)
	}

	if got := stderr.String(); got != "unknown command: unknown\n" {
		t.Errorf("stderr = %q, want %q", got, "unknown command: unknown\n")
	}
}
