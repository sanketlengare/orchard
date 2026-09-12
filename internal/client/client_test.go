package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
	}{
		{
			name:    "url with trailing slash",
			baseURL: "http://example.com/",
		},
		{
			name:    "url without trailing slash",
			baseURL: "http://hello.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			c := New(tt.baseURL)

			if c == nil {
				t.Fatalf("New() returned nil for valid URL")
			}

			if strings.HasSuffix(c.baseURL, "/") {
				t.Errorf("base URL should not have a trailing slash: %q", c.baseURL)
			}

			if c.httpClient.Timeout != 10*time.Second {
				t.Errorf("HTTP client timeout = %v, want %v",
					c.httpClient.Timeout, 10*time.Second)
			}

		})
	}
}

func TestGetJobSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}

		if r.URL.Path != "/v1/jobs/job1" {
			t.Errorf("path = %s, want /v1/jobs/job1", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
            "id": "job1",
            "command": ["echo", "hello"],
            "resources": {
                "cpu_cores": 2,
                "memory_mib": 512
            },
            "status": "running"
        }`))
	}))
	defer server.Close()

	client := New(server.URL)
	got, err := client.GetJob(t.Context(), "job1")
	if err != nil {
		t.Fatalf("GetJob() returned error: %v", err)
	}

	if got.ID != "job1" {
		t.Errorf("ID = %q, want %q", got.ID, "job1")
	}

	if len(got.Command) != 2 ||
		got.Command[0] != "echo" ||
		got.Command[1] != "hello" {
		t.Errorf("Command = %#v, want [echo hello]", got.Command)
	}

	if got.Resources.CPUCores != 2 {
		t.Errorf("CPUCores = %d, want 2", got.Resources.CPUCores)
	}

	if got.Resources.MemoryMiB != 512 {
		t.Errorf("MemoryMiB = %d, want 512", got.Resources.MemoryMiB)
	}

	if got.Status != "running" {
		t.Errorf("Status = %q, want %q", got.Status, "running")
	}
}

func TestGetJobHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	client := New(server.URL)
	_, err := client.GetJob(t.Context(), "job1")

	if err == nil {
		t.Fatal("GetJob() returned nil error, want HTTP failure")
	}

	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error = %q, want it to contain 404", err.Error())
	}
}

func TestGetJobInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":`))
	}))
	defer server.Close()

	client := New(server.URL)
	_, err := client.GetJob(t.Context(), "job1")

	if err == nil {
		t.Fatal("GetJob() returned nil error for malformed JSON")
	}
}

func TestFinishJobSuccess(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		wantStatus int
		wantErr    string
	}{
		{name: "succeeded", status: "succeeded", wantStatus: http.StatusNoContent},
		{name: "failed", status: "failed", wantStatus: http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

				if r.Method != http.MethodPost {
					t.Errorf("method = %s, want POST", r.Method)
				}

				if r.URL.Path != "/v1/jobs/job1/finish" {
					t.Errorf("path = %s, want /v1/jobs/job1/finish", r.URL.Path)
				}

				if got := r.Header.Get("Content-Type"); got != "application/json" {
					t.Errorf("content type = %s, want application/json", got)
				}

				var body JobFinishStatus
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decode request body: %v", err)
					http.Error(w, "invalid JSON", http.StatusBadRequest)
					return
				}

				if body.Status != tt.status {
					t.Errorf("status = %q, want %q", body.Status, tt.status)
				}

				w.WriteHeader(tt.wantStatus)
			}))
			defer server.Close()

			err := New(server.URL).FinishJob(t.Context(), "job1", tt.status)
			if err != nil {
				t.Fatalf("FinishJob() returned error: %v", err)
			}
		})
	}
}

func TestFinishJobHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer server.Close()

	client := New(server.URL)
	err := client.FinishJob(t.Context(), "job1", "failed")

	if err == nil {
		t.Fatal("FinishJob() returned nil error, want HTTP failure")
	}

	if !strings.Contains(err.Error(), "409") {
		t.Errorf("error = %q, want it to contain 409", err.Error())
	}
}
