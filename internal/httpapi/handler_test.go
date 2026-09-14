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
	"github.com/sanketlengare/orchard/internal/worker"
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

// Register worker tests

func TestRegisterWorkerSuccess(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(jobStore, workerStore)

	if err != nil {
		t.Fatalf("NewService() returns error - %v", err)
	}

	handler := NewHandler(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/workers",
		strings.NewReader(`{
			"id": "worker1",
			"capacity": {
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

	var body workerResponse

	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if body.ID != "worker1" {
		t.Fatalf("response ID = %q, want %q", body.ID, "worker1")
	}

	if body.AvailableCapacity.CPUCores != 2 {
		t.Fatalf(
			"response CPU cores = %d, want 2",
			body.AvailableCapacity.CPUCores,
		)
	}

	if body.AvailableCapacity.MemoryMiB != 1024 {
		t.Fatalf(
			"response memory MiB = %d, want 1024",
			body.AvailableCapacity.MemoryMiB,
		)
	}

	if body.TotalCapacity.CPUCores != 2 ||
		body.TotalCapacity.MemoryMiB != 1024 {
		t.Fatalf(
			"total capacity = %v, want {2 1024}",
			body.TotalCapacity,
		)
	}

}

func TestRegisterWorkerMalformed(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(jobStore, workerStore)

	if err != nil {
		t.Fatalf("NewService() returns error - %v", err)
	}

	handler := NewHandler(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/workers",
		strings.NewReader(`{"id": "worker1"`),
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

func TestRegisterWorkerInvalidWorker(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(jobStore, workerStore)

	if err != nil {
		t.Fatalf("NewService() returns error - %v", err)
	}

	handler := NewHandler(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/workers",
		strings.NewReader(`{
		"id": " ",
		"capacity": {
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

func TestRegisterWrongHTTP(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(jobStore, workerStore)

	if err != nil {
		t.Fatalf("NewService() returns error - %v", err)
	}

	handler := NewHandler(service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/v1/workers",
		strings.NewReader(`{
		"id": "worker1",
		"capacity": {
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

func TestRegisterWorkerDuplicate(t *testing.T) {
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
		"id": "worker1",
		"capacity": {
			"cpu_cores": 2,
			"memory_mib": 1024
		}
	}`

	// First request creates the job.

	firstRequest := httptest.NewRequest(
		http.MethodPost,
		"/v1/workers",
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
		"/v1/workers",
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

	workers, err := workerStore.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(workers) != 1 {
		t.Fatalf(
			"stored jobs = %d, want 1",
			len(workers),
		)
	}
}

func TestRegisterWorkerInvalidCapacity(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "zero CPU",
			body: `{
				"id": "worker1",
				"capacity": {
					"cpu_cores": 0,
					"memory_mib": 1024
				}
			}`,
		},
		{
			name: "zero memory",
			body: `{
				"id": "worker1",
				"capacity": {
					"cpu_cores": 2,
					"memory_mib": 0
				}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobStore := store.NewMemoryJobStore()
			workerStore := store.NewMemoryWorkerStore()

			service, err := controlplane.NewService(jobStore, workerStore)
			if err != nil {
				t.Fatalf("NewService() error = %v", err)
			}

			handler := NewHandler(service)

			request := httptest.NewRequest(
				http.MethodPost,
				"/v1/workers",
				strings.NewReader(tt.body),
			)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf(
					"status = %d, want %d; body = %s",
					recorder.Code,
					http.StatusBadRequest,
					recorder.Body.String(),
				)
			}
		})
	}
}

func TestScheduleJobSuccess(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(jobStore, workerStore)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	handler := NewHandler(service)

	worker, errRegisterWorker := service.RegisterWorker("worker1", worker.Capacity{CPUCores: 2, MemoryMiB: 1024})

	if errRegisterWorker != nil {
		t.Fatalf("RegisterWorker() returns error - %v", errRegisterWorker)
	}

	candidateJob, errSubmitJob := service.SubmitJob("job1", []string{"echo", "hello"}, job.ResourceRequest{CPUCores: 2, MemoryMiB: 1024})

	if errSubmitJob != nil {
		t.Fatalf("SubmitJob() returns error - %v", errSubmitJob)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/jobs/job1/schedule",
		strings.NewReader(""),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf(
			"status = %d, want %d",
			response.StatusCode,
			http.StatusOK,
		)
	}

	var body assignmentResponse

	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if body.JobID != candidateJob.ID() {
		t.Fatalf("ID = %q, want %q", body.JobID, candidateJob.ID())
	}

	if body.WorkerID != worker.ID() {
		t.Fatalf("status = %q, want %q", body.WorkerID, worker.ID())
	}

	if candidateJob.Status() != job.StatusRunning {
		t.Fatalf(
			"job status = %v, want %v",
			candidateJob.Status(),
			job.StatusRunning,
		)
	}

}

func TestScheduleJobNotFound(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(jobStore, workerStore)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	handler := NewHandler(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/jobs/job1/schedule",
		strings.NewReader(""),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf(
			"status = %d, want %d",
			response.StatusCode,
			http.StatusNotFound,
		)
	}
}

func TestScheduleJobNoSuitableWorker(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(jobStore, workerStore)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	handler := NewHandler(service)

	_, errRegisterWorker := service.RegisterWorker("worker1", worker.Capacity{CPUCores: 2, MemoryMiB: 1024})

	if errRegisterWorker != nil {
		t.Fatalf("RegisterWorker() returns error - %v", errRegisterWorker)
	}

	_, errSubmitJob := service.SubmitJob("job1", []string{"echo", "hello"}, job.ResourceRequest{CPUCores: 4, MemoryMiB: 2048})

	if errSubmitJob != nil {
		t.Fatalf("SubmitJob() returns error - %v", errSubmitJob)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/jobs/job1/schedule",
		strings.NewReader(""),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusConflict {
		t.Fatalf(
			"status = %d, want %d",
			response.StatusCode,
			http.StatusConflict,
		)
	}
}

func TestScheduleJobNotPending(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(jobStore, workerStore)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	handler := NewHandler(service)

	_, errRegisterWorker := service.RegisterWorker("worker1", worker.Capacity{CPUCores: 4, MemoryMiB: 2048})

	if errRegisterWorker != nil {
		t.Fatalf("RegisterWorker() returns error - %v", errRegisterWorker)
	}

	_, errSubmitJob := service.SubmitJob("job1", []string{"echo", "hello"}, job.ResourceRequest{CPUCores: 2, MemoryMiB: 1024})

	if errSubmitJob != nil {
		t.Fatalf("SubmitJob() returns error - %v", errSubmitJob)
	}

	request1 := httptest.NewRequest(
		http.MethodPost,
		"/v1/jobs/job1/schedule",
		strings.NewReader(""),
	)

	recorder1 := httptest.NewRecorder()

	handler.ServeHTTP(recorder1, request1)

	if recorder1.Code != http.StatusOK {
		t.Fatalf(
			"first status = %d, want %d; body = %s",
			recorder1.Code,
			http.StatusOK,
			recorder1.Body.String(),
		)
	}

	// send request to schedule same job again

	request2 := httptest.NewRequest(
		http.MethodPost,
		"/v1/jobs/job1/schedule",
		strings.NewReader(""),
	)

	recorder2 := httptest.NewRecorder()

	handler.ServeHTTP(recorder2, request2)

	response := recorder2.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusConflict {
		t.Fatalf(
			"status = %d, want %d",
			response.StatusCode,
			http.StatusConflict,
		)
	}
}

func TestFinishJobSuccessSucceededStatus(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(jobStore, workerStore)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	handler := NewHandler(service)

	registeredWorker, errRegisterWorker := service.RegisterWorker("worker1", worker.Capacity{CPUCores: 4, MemoryMiB: 2048})

	if errRegisterWorker != nil {
		t.Fatalf("RegisterWorker() returns error - %v", errRegisterWorker)
	}

	j, errSubmitJob := service.SubmitJob("job1", []string{"echo", "hello"}, job.ResourceRequest{CPUCores: 2, MemoryMiB: 1024})

	if errSubmitJob != nil {
		t.Fatalf("SubmitJob() returns error - %v", errSubmitJob)
	}

	_, err = service.ScheduleJob(j.ID())

	if err != nil {
		t.Fatalf("ScheduleJob() error = %v", err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/jobs/job1/finish",
		strings.NewReader(`{
			"status": "succeeded"
		}`),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		t.Fatalf(
			"status = %d, want %d",
			response.StatusCode,
			http.StatusNoContent,
		)
	}

	if j.Status() != job.StatusSucceeded {
		t.Fatalf("Job status is - %v, want - %v", j.Status(), job.StatusSucceeded)
	}

	if recorder.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", recorder.Body.String())
	}

	if got := registeredWorker.AvailableCapacity(); got != registeredWorker.TotalCapacity() {
		t.Fatalf(
			"available capacity = %v, want %v",
			got,
			registeredWorker.TotalCapacity(),
		)
	}

}

func TestFinishJobSuccessFailedStatus(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(jobStore, workerStore)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	handler := NewHandler(service)

	registeredWorker, errRegisterWorker := service.RegisterWorker("worker1", worker.Capacity{CPUCores: 4, MemoryMiB: 2048})

	if errRegisterWorker != nil {
		t.Fatalf("RegisterWorker() returns error - %v", errRegisterWorker)
	}

	j, errSubmitJob := service.SubmitJob("job1", []string{"echo", "hello"}, job.ResourceRequest{CPUCores: 2, MemoryMiB: 1024})

	if errSubmitJob != nil {
		t.Fatalf("SubmitJob() returns error - %v", errSubmitJob)
	}

	_, err = service.ScheduleJob(j.ID())

	if err != nil {
		t.Fatalf("ScheduleJob() error = %v", err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/jobs/job1/finish",
		strings.NewReader(`{
			"status": "failed"
		}`),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		t.Fatalf(
			"status = %d, want %d",
			response.StatusCode,
			http.StatusNoContent,
		)
	}

	if j.Status() != job.StatusFailed {
		t.Fatalf("Job status is - %v, want - %v", j.Status(), job.StatusFailed)
	}

	if recorder.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", recorder.Body.String())
	}

	if got := registeredWorker.AvailableCapacity(); got != registeredWorker.TotalCapacity() {
		t.Fatalf(
			"available capacity = %v, want %v",
			got,
			registeredWorker.TotalCapacity(),
		)
	}

}

func TestFinishJobFailsInvalidStatus(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(jobStore, workerStore)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	handler := NewHandler(service)

	_, errRegisterWorker := service.RegisterWorker("worker1", worker.Capacity{CPUCores: 4, MemoryMiB: 2048})

	if errRegisterWorker != nil {
		t.Fatalf("RegisterWorker() returns error - %v", errRegisterWorker)
	}

	j, errSubmitJob := service.SubmitJob("job1", []string{"echo", "hello"}, job.ResourceRequest{CPUCores: 2, MemoryMiB: 1024})

	if errSubmitJob != nil {
		t.Fatalf("SubmitJob() returns error - %v", errSubmitJob)
	}

	_, err = service.ScheduleJob(j.ID())

	if err != nil {
		t.Fatalf("ScheduleJob() error = %v", err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/jobs/job1/finish",
		strings.NewReader(`{
			"status": "running"
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

	if j.Status() != job.StatusRunning {
		t.Fatalf("Job status is - %v, want - %v", j.Status(), job.StatusRunning)
	}

}

func TestFinishJobFailsJobNotFound(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(jobStore, workerStore)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	handler := NewHandler(service)

	_, errRegisterWorker := service.RegisterWorker("worker1", worker.Capacity{CPUCores: 4, MemoryMiB: 2048})

	if errRegisterWorker != nil {
		t.Fatalf("RegisterWorker() returns error - %v", errRegisterWorker)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/jobs/job1/finish",
		strings.NewReader(`{
			"status": "succeeded"
		}`),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf(
			"status = %d, want %d",
			response.StatusCode,
			http.StatusNotFound,
		)
	}

}

func TestFinishJobFailsAssignmentNotFound(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(jobStore, workerStore)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	handler := NewHandler(service)

	_, errRegisterWorker := service.RegisterWorker("worker1", worker.Capacity{CPUCores: 4, MemoryMiB: 2048})

	if errRegisterWorker != nil {
		t.Fatalf("RegisterWorker() returns error - %v", errRegisterWorker)
	}

	_, errSubmitJob := service.SubmitJob("job1", []string{"echo", "hello"}, job.ResourceRequest{CPUCores: 2, MemoryMiB: 1024})

	if errSubmitJob != nil {
		t.Fatalf("SubmitJob() returns error - %v", errSubmitJob)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/jobs/job1/finish",
		strings.NewReader(`{
			"status": "succeeded"
		}`),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusConflict {
		t.Fatalf(
			"status = %d, want %d",
			response.StatusCode,
			http.StatusConflict,
		)
	}

}

func TestFinishJobFailsFinishTwice(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(jobStore, workerStore)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	handler := NewHandler(service)

	_, errRegisterWorker := service.RegisterWorker("worker1", worker.Capacity{CPUCores: 4, MemoryMiB: 2048})

	if errRegisterWorker != nil {
		t.Fatalf("RegisterWorker() returns error - %v", errRegisterWorker)
	}

	_, errSubmitJob := service.SubmitJob("job1", []string{"echo", "hello"}, job.ResourceRequest{CPUCores: 2, MemoryMiB: 1024})

	if errSubmitJob != nil {
		t.Fatalf("SubmitJob() returns error - %v", errSubmitJob)
	}

	_, err = service.ScheduleJob("job1")

	if err != nil {
		t.Fatalf("ScheduleJob() returns error - %v", err)
	}

	request1 := httptest.NewRequest(
		http.MethodPost,
		"/v1/jobs/job1/finish",
		strings.NewReader(`{
		"status": "succeeded"
		}`),
	)

	recorder1 := httptest.NewRecorder()

	handler.ServeHTTP(recorder1, request1)

	if recorder1.Code != http.StatusNoContent {
		t.Fatalf(
			"first status = %d, want %d; body = %s",
			recorder1.Code,
			http.StatusNoContent,
			recorder1.Body.String(),
		)
	}

	// send request to schedule same job again

	request2 := httptest.NewRequest(
		http.MethodPost,
		"/v1/jobs/job1/finish",
		strings.NewReader(`{
		"status": "succeeded"
		}`),
	)

	recorder2 := httptest.NewRecorder()

	handler.ServeHTTP(recorder2, request2)

	response := recorder2.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusConflict {
		t.Fatalf(
			"status = %d, want %d",
			response.StatusCode,
			http.StatusConflict,
		)
	}
}

func TestGetAssignmentSuccess(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(jobStore, workerStore)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	handler := NewHandler(service)

	_, errRegisterWorker := service.RegisterWorker("worker1", worker.Capacity{CPUCores: 4, MemoryMiB: 2048})

	if errRegisterWorker != nil {
		t.Fatalf("RegisterWorker() returns error - %v", errRegisterWorker)
	}

	_, errSubmitJob := service.SubmitJob("job1", []string{"echo", "hello"}, job.ResourceRequest{CPUCores: 2, MemoryMiB: 1024})

	if errSubmitJob != nil {
		t.Fatalf("SubmitJob() returns error - %v", errSubmitJob)
	}

	_, err = service.ScheduleJob("job1")

	if err != nil {
		t.Fatalf("ScheduleJob() returns error - %v", err)
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"/v1/jobs/job1/assignment",
		strings.NewReader(""),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf(
			"status = %d, want %d",
			response.StatusCode,
			http.StatusOK,
		)
	}

	var body assignmentResponse

	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if body.JobID != "job1" {
		t.Fatalf("ID = %q, want %q", body.JobID, "job1")
	}

	if body.WorkerID != "worker1" {
		t.Fatalf("status = %q, want %q", body.WorkerID, "worker1")
	}
}

func TestGetAssignmentFail(t *testing.T) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(jobStore, workerStore)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	handler := NewHandler(service)

	_, errRegisterWorker := service.RegisterWorker("worker1", worker.Capacity{CPUCores: 4, MemoryMiB: 2048})

	if errRegisterWorker != nil {
		t.Fatalf("RegisterWorker() returns error - %v", errRegisterWorker)
	}

	_, errSubmitJob := service.SubmitJob("job1", []string{"echo", "hello"}, job.ResourceRequest{CPUCores: 2, MemoryMiB: 1024})

	if errSubmitJob != nil {
		t.Fatalf("SubmitJob() returns error - %v", errSubmitJob)
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"/v1/jobs/job1/assignment",
		strings.NewReader(""),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf(
			"status = %d, want %d",
			response.StatusCode,
			http.StatusNotFound,
		)
	}

	var body errorResponse

	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if body.Error != controlplane.ErrAssignmentNotFound.Error() {
		t.Fatalf(
			"error = %q, want %q",
			body.Error,
			controlplane.ErrAssignmentNotFound.Error(),
		)
	}
}
