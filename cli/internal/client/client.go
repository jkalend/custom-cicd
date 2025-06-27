package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client represents the API client for the CI/CD system
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new API client
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Pipeline represents a pipeline configuration
type Pipeline struct {
	ID         string   `json:"id,omitempty"`
	Name       string   `json:"name"`
	Steps      []Step   `json:"steps"`
	Status     string   `json:"status,omitempty"`
	CreatedAt  string   `json:"created_at,omitempty"`
	StartedAt  *string  `json:"started_at,omitempty"`
	FinishedAt *string  `json:"finished_at,omitempty"`
	Duration   *float64 `json:"total_duration,omitempty"`
}

// Step represents a pipeline step
type Step struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Status  string `json:"status,omitempty"`
	Output  string `json:"output,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Run represents a pipeline run
type Run struct {
	ID         string   `json:"id"`
	PipelineID string   `json:"pipeline_id"`
	Name       string   `json:"name"`
	Status     string   `json:"status"`
	CreatedAt  string   `json:"created_at"`
	StartedAt  *string  `json:"started_at,omitempty"`
	FinishedAt *string  `json:"finished_at,omitempty"`
	Duration   *float64 `json:"total_duration,omitempty"`
	Steps      []Step   `json:"steps,omitempty"`
}

// CreatePipelineResponse represents the response from creating a pipeline
type CreatePipelineResponse struct {
	PipelineID string `json:"pipeline_id"`
	Status     string `json:"status"`
}

// CreateAndRunResponse represents the response from creating and running a pipeline
type CreateAndRunResponse struct {
	PipelineID string `json:"pipeline_id"`
	RunID      string `json:"run_id"`
	Status     string `json:"status"`
}

// RunPipelineResponse represents the response from running a pipeline
type RunPipelineResponse struct {
	RunID  string `json:"run_id"`
	Status string `json:"status"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status      string `json:"status"`
	Timestamp   string `json:"timestamp"`
	AgentStatus string `json:"agent_status"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error string `json:"error"`
}

// APIResponse represents the wrapped response from the backend
type APIResponse struct {
	Data    interface{} `json:"data"`
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
}

// doRequest performs an HTTP request and handles the response
func (c *Client) doRequest(method, endpoint string, body interface{}, response interface{}) error {
	url := c.BaseURL + endpoint

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errorResp ErrorResponse
		if err := json.Unmarshal(respBody, &errorResp); err != nil {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}
		return fmt.Errorf("API error: %s", errorResp.Error)
	}

	if response != nil {
		// Try to unmarshal as wrapped API response first
		var apiResp APIResponse
		if err := json.Unmarshal(respBody, &apiResp); err == nil {
			if !apiResp.Success {
				return fmt.Errorf("API error: %s", apiResp.Error)
			}
			// Marshal the data field and unmarshal into the target response
			dataBytes, err := json.Marshal(apiResp.Data)
			if err != nil {
				return fmt.Errorf("failed to marshal API data: %w", err)
			}
			if err := json.Unmarshal(dataBytes, response); err != nil {
				return fmt.Errorf("failed to unmarshal API data: %w", err)
			}
		} else {
			// Fallback to direct unmarshaling for non-wrapped responses
			if err := json.Unmarshal(respBody, response); err != nil {
				return fmt.Errorf("failed to unmarshal response: %w", err)
			}
		}
	}

	return nil
}

// HealthCheck checks the health of the API
func (c *Client) HealthCheck() (*HealthResponse, error) {
	var response HealthResponse
	err := c.doRequest("GET", "/health", nil, &response)
	return &response, err
}

// CreatePipeline creates a new pipeline
func (c *Client) CreatePipeline(pipeline *Pipeline) (*CreatePipelineResponse, error) {
	var response CreatePipelineResponse
	err := c.doRequest("POST", "/api/pipelines", pipeline, &response)
	return &response, err
}

// CreateAndRunPipeline creates and immediately runs a pipeline
func (c *Client) CreateAndRunPipeline(pipeline *Pipeline) (*CreateAndRunResponse, error) {
	var response CreateAndRunResponse
	err := c.doRequest("POST", "/api/pipelines/run", pipeline, &response)
	return &response, err
}

// ListPipelines lists all pipelines
func (c *Client) ListPipelines() ([]Pipeline, error) {
	var pipelines []Pipeline
	err := c.doRequest("GET", "/api/pipelines", nil, &pipelines)
	return pipelines, err
}

// GetPipeline gets a specific pipeline by ID
func (c *Client) GetPipeline(pipelineID string) (*Pipeline, error) {
	var pipeline Pipeline
	endpoint := fmt.Sprintf("/api/pipelines/%s", pipelineID)
	err := c.doRequest("GET", endpoint, nil, &pipeline)
	return &pipeline, err
}

// RunPipeline runs an existing pipeline
func (c *Client) RunPipeline(pipelineID string, background bool) (*RunPipelineResponse, error) {
	var response RunPipelineResponse
	endpoint := fmt.Sprintf("/api/pipelines/%s/run?background=%t", pipelineID, background)
	err := c.doRequest("POST", endpoint, nil, &response)
	return &response, err
}

// CancelPipeline cancels a running pipeline
func (c *Client) CancelPipeline(pipelineID string) error {
	endpoint := fmt.Sprintf("/api/pipelines/%s/cancel", pipelineID)
	return c.doRequest("POST", endpoint, nil, nil)
}

// DeletePipeline deletes a pipeline
func (c *Client) DeletePipeline(pipelineID string) error {
	endpoint := fmt.Sprintf("/api/pipelines/%s", pipelineID)
	return c.doRequest("DELETE", endpoint, nil, nil)
}

// ListRuns lists all runs, optionally filtered by pipeline ID
func (c *Client) ListRuns(pipelineID string) ([]Run, error) {
	var runs []Run
	endpoint := "/api/runs"
	if pipelineID != "" {
		endpoint += "?pipeline_id=" + url.QueryEscape(pipelineID)
	}
	err := c.doRequest("GET", endpoint, nil, &runs)
	return runs, err
}

// GetRun gets a specific run by ID
func (c *Client) GetRun(runID string) (*Run, error) {
	var run Run
	endpoint := fmt.Sprintf("/api/runs/%s", runID)
	err := c.doRequest("GET", endpoint, nil, &run)
	return &run, err
}

// CancelRun cancels a running pipeline run
func (c *Client) CancelRun(runID string) error {
	endpoint := fmt.Sprintf("/api/runs/%s/cancel", runID)
	return c.doRequest("POST", endpoint, nil, nil)
}

// DeleteRun deletes a specific run
func (c *Client) DeleteRun(runID string) error {
	endpoint := fmt.Sprintf("/api/runs/%s", runID)
	return c.doRequest("DELETE", endpoint, nil, nil)
}
