package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestRunServer(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	called := false

	serve := func(stdout io.Writer) error {
		called = true
		return nil
	}

	code := runWithServer([]string{"server"}, &stdout, &stderr, serve)

	if code != 0 {
		t.Fatalf("code - %d, want 0", code)
	}

	if called != true {
		t.Fatalf("called - %v, want true", called)
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty output", got)
	}

}

func TestRunServerError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	serve := func(stdout io.Writer) error {
		return errors.New("listen failed")
	}

	code := runWithServer([]string{"server"}, &stdout, &stderr, serve)

	if code != 1 {
		t.Fatalf("code - %d, want 1", code)
	}

	if got := stderr.String(); got != "server error: listen failed\n" {
		t.Fatalf("stderr = %q, want empty - listen failed", got)
	}
}

func TestNewHTTPHandler(t *testing.T) {
	handler, err := newHTTPHandler()

	if err != nil {
		t.Fatalf("newHTTPHandler() returns error - %v", err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/jobs",
		strings.NewReader(`{
		"id": "job1",
		"command": ["echo", "hello"],
		"resources": {
			"cpu_cores": 2,
			"memory_mib": 1024
		}
	}`),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf(
			"status = %d, want %d",
			response.StatusCode,
			http.StatusCreated,
		)
	}
}
