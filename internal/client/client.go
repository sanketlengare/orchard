package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Job struct {
	ID        string    `json:"id"`
	Command   []string  `json:"command"`
	Resources Resources `json:"resources"`
	Status    string    `json:"status"`
}

type Resources struct {
	CPUCores  int `json:"cpu_cores"`
	MemoryMiB int `json:"memory_mib"`
}

type JobFinishStatus struct {
	Status string `json:"status"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type Assignment struct {
	JobID    string `json:"job_id"`
	WorkerID string `json:"worker_id"`
}

func New(baseURL string) *Client {

	baseURL = strings.TrimRight(baseURL, "/")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: client,
	}
}

func (c *Client) GetJob(ctx context.Context, id string) (Job, error) {
	urlGet := c.baseURL + "/v1/jobs/" + url.PathEscape(id)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, urlGet, nil)

	if err != nil {
		return Job{}, err
	}

	response, err := c.httpClient.Do(request)

	if err != nil {
		return Job{}, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return Job{}, fmt.Errorf("get job failed: %s", response.Status)
	}

	var jobResponse Job

	if err := json.NewDecoder(response.Body).Decode(&jobResponse); err != nil {
		return Job{}, fmt.Errorf("job decode failed: %w", err)
	}

	return jobResponse, nil
}

func (c *Client) GetAssignment(ctx context.Context, id string) (Assignment, error) {
	urlGetAssignment := c.baseURL + "/v1/jobs/" + url.PathEscape(id) + "/assignment"

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, urlGetAssignment, nil)

	if err != nil {
		return Assignment{}, err
	}

	response, err := c.httpClient.Do(request)

	if err != nil {
		return Assignment{}, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return Assignment{}, fmt.Errorf("get assignment failed: %s", response.Status)
	}

	var assignmentResponse Assignment

	if err := json.NewDecoder(response.Body).Decode(&assignmentResponse); err != nil {
		return Assignment{}, fmt.Errorf("assignment decode failed: %w", err)
	}

	return assignmentResponse, nil
}

func (c *Client) FinishJob(ctx context.Context, id string, status string) error {
	urlFinishJob := c.baseURL + "/v1/jobs/" + url.PathEscape(id) + "/finish"

	finishStatus := JobFinishStatus{
		Status: status,
	}

	body, err := json.Marshal(finishStatus)

	if err != nil {
		return fmt.Errorf("json marshal failed with error - %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, urlFinishJob, bytes.NewReader(body))

	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)

	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("finish job failed: %d", response.StatusCode)
	}

	return nil
}
