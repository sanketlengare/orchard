package client

import (
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

type Client struct {
	baseURL    string
	httpClient *http.Client
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
