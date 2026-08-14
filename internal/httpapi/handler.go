package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/sanketlengare/orchard/internal/job"
	"github.com/sanketlengare/orchard/internal/store"
)

type JobSubmitter interface {
	SubmitJob(
		id string,
		command []string,
		resources job.ResourceRequest,
	) (*job.Job, error)
}

type Handler struct {
	service JobSubmitter
	mux     *http.ServeMux
}

func NewHandler(service JobSubmitter) *Handler {
	handler := &Handler{
		service: service,
		mux:     http.NewServeMux(),
	}

	handler.mux.HandleFunc(
		"POST /v1/jobs",
		handler.submitJob,
	)

	return handler
}

func (h *Handler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.mux.ServeHTTP(w, r)
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

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) submitJob(
	w http.ResponseWriter,
	r *http.Request,
) {
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

	writeJSON(
		w,
		http.StatusCreated,
		response,
	)
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
