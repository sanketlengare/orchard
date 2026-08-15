package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sanketlengare/orchard/internal/controlplane"
	"github.com/sanketlengare/orchard/internal/job"
	"github.com/sanketlengare/orchard/internal/store"
)

func TestSubmitJobSuccess(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(jobStore, workerStore)

	if err != nil {
		t.Fatalf("NewService() returns error - %v", err)
	}

	handler := NewHandler(service)

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

	if contentType := response.Header.Get("Content-Type"); contentType != "application/json" {
		t.Fatalf(
			"Content-Type = %q, want %q",
			contentType,
			"application/json",
		)
	}

	var body jobResponse

	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if body.ID != "job1" {
		t.Fatalf("response ID = %q, want %q", body.ID, "job1")
	}

	if len(body.Command) != 2 ||
		body.Command[0] != "echo" ||
		body.Command[1] != "hello" {
		t.Fatalf(
			"response command = %v, want %v",
			body.Command,
			[]string{"echo", "hello"},
		)
	}

	if body.Resources.CPUCores != 2 {
		t.Fatalf(
			"response CPU cores = %d, want 2",
			body.Resources.CPUCores,
		)
	}

	if body.Resources.MemoryMiB != 1024 {
		t.Fatalf(
			"response memory MiB = %d, want 1024",
			body.Resources.MemoryMiB,
		)
	}

	if body.Status != "pending" {
		t.Fatalf(
			"response status = %q, want %q",
			body.Status,
			"pending",
		)
	}
}

func TestSubmitJobMalformed(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(jobStore, workerStore)

	if err != nil {
		t.Fatalf("NewService() returns error - %v", err)
	}

	handler := NewHandler(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/jobs",
		strings.NewReader(`{"id": "job1"`),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf(
			"status = %d, want %d",
			response.StatusCode,
			http.StatusBadRequest,
		)
	}

	if contentType := response.Header.Get("Content-Type"); contentType != "application/json" {
		t.Fatalf(
			"Content-Type = %q, want %q",
			contentType,
			"application/json",
		)
	}

	var body errorResponse

	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if body.Error != "invalid request body" {
		t.Fatalf(
			"error = %q, want %q",
			body.Error,
			"invalid request body",
		)
	}
}

func TestSubmitJobInvalidJob(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(jobStore, workerStore)

	if err != nil {
		t.Fatalf("NewService() returns error - %v", err)
	}

	handler := NewHandler(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/jobs",
		strings.NewReader(`{
		"id": " ",
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

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf(
			"status = %d, want %d",
			response.StatusCode,
			http.StatusBadRequest,
		)
	}
}

func TestSubmitWrongHTTP(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(jobStore, workerStore)

	if err != nil {
		t.Fatalf("NewService() returns error - %v", err)
	}

	handler := NewHandler(service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/v1/jobs",
		strings.NewReader(`{
		"id": " ",
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

	if response.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf(
			"status = %d, want %d",
			response.StatusCode,
			http.StatusMethodNotAllowed,
		)
	}
}

func TestSubmitJobDuplicate(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(
		jobStore,
		workerStore,
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	handler := NewHandler(service)

	const requestBody = `{
		"id": "job1",
		"command": ["echo", "hello"],
		"resources": {
			"cpu_cores": 2,
			"memory_mib": 1024
		}
	}`

	// First request creates the job.

	firstRequest := httptest.NewRequest(
		http.MethodPost,
		"/v1/jobs",
		strings.NewReader(requestBody),
	)

	firstRecorder := httptest.NewRecorder()

	handler.ServeHTTP(
		firstRecorder,
		firstRequest,
	)

	firstResponse := firstRecorder.Result()
	defer firstResponse.Body.Close()

	if firstResponse.StatusCode != http.StatusCreated {
		t.Fatalf(
			"first status = %d, want %d",
			firstResponse.StatusCode,
			http.StatusCreated,
		)
	}

	// Second request uses the same ID and same store.

	secondRequest := httptest.NewRequest(
		http.MethodPost,
		"/v1/jobs",
		strings.NewReader(requestBody),
	)

	secondRecorder := httptest.NewRecorder()

	handler.ServeHTTP(
		secondRecorder,
		secondRequest,
	)

	secondResponse := secondRecorder.Result()
	defer secondResponse.Body.Close()

	if secondResponse.StatusCode != http.StatusConflict {
		t.Fatalf(
			"second status = %d, want %d",
			secondResponse.StatusCode,
			http.StatusConflict,
		)
	}

	jobs, err := jobStore.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf(
			"stored jobs = %d, want 1",
			len(jobs),
		)
	}
}

func TestGetJob(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(
		jobStore,
		workerStore,
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	handler := NewHandler(service)

	firstRequest := httptest.NewRequest(
		http.MethodGet,
		"/v1/jobs/job1",
		strings.NewReader(""),
	)

	firstRecorder := httptest.NewRecorder()

	handler.ServeHTTP(
		firstRecorder,
		firstRequest,
	)

	firstResponse := firstRecorder.Result()
	defer firstResponse.Body.Close()

	if firstResponse.StatusCode != http.StatusNotFound {
		t.Fatalf(
			"first status = %d, want %d",
			firstResponse.StatusCode,
			http.StatusNotFound,
		)
	}

	var errorBody errorResponse

	if err := json.NewDecoder(firstResponse.Body).Decode(&errorBody); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if errorBody.Error != store.ErrJobNotFound.Error() {
		t.Fatalf(
			"error = %q, want %q",
			errorBody.Error,
			store.ErrJobNotFound.Error(),
		)
	}

	// submit a job directly to jobstore

	_, err = service.SubmitJob("job1", []string{"echo", "hello"}, job.ResourceRequest{CPUCores: 4, MemoryMiB: 4096})

	if err != nil {
		t.Fatalf("SubmitJob() returns error - %v", err)
	}

	secondRequest := httptest.NewRequest(
		http.MethodGet,
		"/v1/jobs/job1",
		strings.NewReader(""),
	)

	secondRecorder := httptest.NewRecorder()

	handler.ServeHTTP(
		secondRecorder,
		secondRequest,
	)

	secondResponse := secondRecorder.Result()
	defer secondResponse.Body.Close()

	if secondResponse.StatusCode != http.StatusOK {
		t.Fatalf(
			"second status = %d, want %d",
			secondResponse.StatusCode,
			http.StatusOK,
		)
	}

	var body jobResponse

	if err := json.NewDecoder(secondResponse.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if body.ID != "job1" {
		t.Fatalf("ID = %q, want %q", body.ID, "job1")
	}

	if body.Status != "pending" {
		t.Fatalf("status = %q, want %q", body.Status, "pending")
	}
}
