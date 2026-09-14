package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/sanketlengare/orchard/internal/controlplane"
	"github.com/sanketlengare/orchard/internal/job"
	"github.com/sanketlengare/orchard/internal/scheduler"
	"github.com/sanketlengare/orchard/internal/store"
	"github.com/sanketlengare/orchard/internal/worker"
)

type ControlPlaneService interface {
	SubmitJob(id string, command []string, resources job.ResourceRequest) (*job.Job, error)
	GetJob(id string) (*job.Job, error)
	RegisterWorker(id string, capacity worker.Capacity) (*worker.Worker, error)
	ScheduleJob(jobID string) (*worker.Worker, error)
	FinishJob(jobID string, finalStatus job.Status) error
	AssignedWorker(jobID string) (*worker.Worker, error)
}

type Handler struct {
	service ControlPlaneService
	mux     *http.ServeMux
}

func NewHandler(service ControlPlaneService) *Handler {
	handler := &Handler{
		service: service,
		mux:     http.NewServeMux(),
	}

	handler.mux.HandleFunc("POST /v1/jobs", handler.submitJob)
	handler.mux.HandleFunc("GET /v1/jobs/{id}", handler.getJob)
	handler.mux.HandleFunc("POST /v1/workers", handler.registerWorker)
	handler.mux.HandleFunc("POST /v1/jobs/{id}/schedule", handler.scheduleJob)
	handler.mux.HandleFunc("POST /v1/jobs/{id}/finish", handler.finishJob)
	handler.mux.HandleFunc("GET /v1/jobs/{id}/assignment", handler.getAssignment)

	return handler
}

func (h *Handler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.mux.ServeHTTP(w, r)
}

type finishJobRequest struct {
	Status string `json:"status"`
}

type submitJobRequest struct {
	ID        string          `json:"id"`
	Command   []string        `json:"command"`
	Resources resourceRequest `json:"resources"`
}

type resourceRequest struct {
	CPUCores  int `json:"cpu_cores"`
	MemoryMiB int `json:"memory_mib"`
}

type assignmentResponse struct {
	JobID    string `json:"job_id"`
	WorkerID string `json:"worker_id"`
}

type jobResponse struct {
	ID        string           `json:"id"`
	Command   []string         `json:"command"`
	Resources resourceResponse `json:"resources"`
	Status    string           `json:"status"`
}

type resourceResponse struct {
	CPUCores  int `json:"cpu_cores"`
	MemoryMiB int `json:"memory_mib"`
}

type registerWorkerRequest struct {
	ID       string          `json:"id"`
	Capacity capacityRequest `json:"capacity"`
}

type capacityRequest struct {
	CPUCores  int `json:"cpu_cores"`
	MemoryMiB int `json:"memory_mib"`
}

type workerResponse struct {
	ID                string           `json:"id"`
	TotalCapacity     capacityResponse `json:"total_capacity"`
	AvailableCapacity capacityResponse `json:"available_capacity"`
}

type capacityResponse struct {
	CPUCores  int `json:"cpu_cores"`
	MemoryMiB int `json:"memory_mib"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) getJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	job, err := h.service.GetJob(id)

	if err != nil {
		status := getJobErrorStatus(err)
		message := err.Error()

		if status == http.StatusInternalServerError {
			message = "internal server error"
		}

		writeJSON(
			w,
			status,
			errorResponse{
				Error: message,
			},
		)
		return
	}

	response := mapJobToResponse(job)

	writeJSON(
		w,
		http.StatusOK,
		response,
	)

}

func (h *Handler) registerWorker(w http.ResponseWriter, r *http.Request) {
	var request registerWorkerRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			errorResponse{
				Error: "invalid request body",
			},
		)
		return
	}

	created, err := h.service.RegisterWorker(request.ID, worker.Capacity(request.Capacity))

	if err != nil {
		status := getRegisterWorkerErrorStatus(err)
		message := err.Error()

		if status == http.StatusInternalServerError {
			message = "internal server error"
		}

		writeJSON(
			w,
			status,
			errorResponse{
				Error: message,
			},
		)
		return
	}

	response := mapWorkerToResponse(created)

	writeJSON(
		w,
		http.StatusCreated,
		response,
	)
}

func (h *Handler) submitJob(w http.ResponseWriter, r *http.Request) {

	var request submitJobRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			errorResponse{
				Error: "invalid request body",
			},
		)
		return
	}

	created, err := h.service.SubmitJob(
		request.ID,
		request.Command,
		job.ResourceRequest{
			CPUCores:  request.Resources.CPUCores,
			MemoryMiB: request.Resources.MemoryMiB,
		},
	)
	if err != nil {
		status := submitJobErrorStatus(err)
		message := err.Error()

		if status == http.StatusInternalServerError {
			message = "internal server error"
		}

		writeJSON(
			w,
			status,
			errorResponse{
				Error: message,
			},
		)
		return
	}

	response := mapJobToResponse(created)

	writeJSON(
		w,
		http.StatusCreated,
		response,
	)
}

func (h *Handler) scheduleJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	worker, err := h.service.ScheduleJob(id)

	if err != nil {
		status := getScheduleJobErrorStatus(err)
		message := err.Error()

		if status == http.StatusInternalServerError {
			message = "internal server error"
		}

		writeJSON(
			w,
			status,
			errorResponse{
				Error: message,
			},
		)
		return
	}

	response := assignmentResponse{
		JobID:    id,
		WorkerID: worker.ID(),
	}

	writeJSON(
		w,
		http.StatusOK,
		response,
	)
}

func (h *Handler) finishJob(w http.ResponseWriter, r *http.Request) {
	var request finishJobRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			errorResponse{
				Error: "invalid request body",
			},
		)
		return
	}

	id := r.PathValue("id")

	err := h.service.FinishJob(id, job.Status(request.Status))

	if err != nil {
		status := getFinishJobErrorStatus(err)
		message := err.Error()

		if status == http.StatusInternalServerError {
			message = "internal server error"
		}

		writeJSON(
			w,
			status,
			errorResponse{
				Error: message,
			},
		)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getAssignment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	assignedWorker, err := h.service.AssignedWorker(id)

	if err != nil {
		status := getAssignmentErrorStatus(err)
		message := err.Error()

		if status == http.StatusInternalServerError {
			message = "internal server error"
		}

		writeJSON(
			w,
			status,
			errorResponse{
				Error: message,
			},
		)

		return
	}

	response := assignmentResponse{
		JobID:    id,
		WorkerID: assignedWorker.ID(),
	}

	writeJSON(
		w,
		http.StatusOK,
		response,
	)
}

func mapWorkerToResponse(created *worker.Worker) workerResponse {
	totalCapacity := created.TotalCapacity()
	availableCapcity := created.AvailableCapacity()

	response := workerResponse{
		ID: created.ID(),
		TotalCapacity: capacityResponse{
			CPUCores:  totalCapacity.CPUCores,
			MemoryMiB: totalCapacity.MemoryMiB,
		},
		AvailableCapacity: capacityResponse{
			CPUCores:  availableCapcity.CPUCores,
			MemoryMiB: availableCapcity.MemoryMiB,
		},
	}

	return response
}

func mapJobToResponse(created *job.Job) jobResponse {
	resources := created.Resources()

	response := jobResponse{
		ID:      created.ID(),
		Command: created.Command(),
		Resources: resourceResponse{
			CPUCores:  resources.CPUCores,
			MemoryMiB: resources.MemoryMiB,
		},
		Status: string(created.Status()),
	}

	return response
}

func getJobErrorStatus(err error) int {
	switch {
	case errors.Is(err, store.ErrJobNotFound):
		return http.StatusNotFound

	default:
		return http.StatusInternalServerError
	}
}

func submitJobErrorStatus(err error) int {
	switch {
	case errors.Is(err, job.ErrIDRequired),
		errors.Is(err, job.ErrCommandRequired),
		errors.Is(err, job.ErrCPUCoresRequired),
		errors.Is(err, job.ErrMemoryMiBRequired):
		return http.StatusBadRequest

	case errors.Is(err, store.ErrJobAlreadyExists):
		return http.StatusConflict

	default:
		return http.StatusInternalServerError
	}
}

func getRegisterWorkerErrorStatus(err error) int {
	switch {
	case errors.Is(err, store.ErrWorkerAlreadyExists):
		return http.StatusConflict

	case errors.Is(err, worker.ErrIDRequired),
		errors.Is(err, worker.ErrCPUCoresRequired),
		errors.Is(err, worker.ErrMemoryMiBRequired):
		return http.StatusBadRequest

	default:
		return http.StatusInternalServerError
	}
}

func getScheduleJobErrorStatus(err error) int {
	switch {
	case errors.Is(err, store.ErrJobNotFound):
		return http.StatusNotFound

	case errors.Is(err, scheduler.ErrNoSuitableWorker),
		errors.Is(err, scheduler.ErrJobNotPending):
		return http.StatusConflict

	default:
		return http.StatusInternalServerError
	}
}

func getFinishJobErrorStatus(err error) int {
	switch {
	case errors.Is(err, scheduler.ErrInvalidFinalStatus):
		return http.StatusBadRequest

	case errors.Is(err, store.ErrJobNotFound):
		return http.StatusNotFound

	case errors.Is(err, controlplane.ErrAssignmentNotFound),
		errors.Is(err, scheduler.ErrJobNotRunning):
		return http.StatusConflict

	default:
		return http.StatusInternalServerError
	}
}

func getAssignmentErrorStatus(err error) int {
	switch {
	case errors.Is(err, controlplane.ErrAssignmentNotFound):
		return http.StatusNotFound

	default:
		return http.StatusInternalServerError
	}
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	value any,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(value)
}
