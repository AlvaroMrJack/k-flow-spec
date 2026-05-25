package kapso

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) do(ctx context.Context, method, path string, body interface{}, out interface{}) error {
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		apiErr := &APIError{
			StatusCode: res.StatusCode,
			Message:    fmt.Sprintf("HTTP %d", res.StatusCode),
		}

		var errBody struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(res.Body).Decode(&errBody); err == nil && errBody.Error != "" {
			apiErr.Message = errBody.Error
		}

		if retryAfter := res.Header.Get("Retry-After"); retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				apiErr.RetryAfter = seconds
			}
		}

		switch res.StatusCode {
		case 429:
			return &RateLimitError{APIError: *apiErr}
		case 404:
			return &NotFoundError{APIError: *apiErr}
		case 422:
			return &ValidationError{APIError: *apiErr}
		default:
			return apiErr
		}
	}

	if out != nil {
		if err := json.NewDecoder(res.Body).Decode(out); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) ListWorkflows(ctx context.Context) ([]Workflow, error) {
	var out struct {
		Data []Workflow `json:"data"`
	}
	err := c.do(ctx, "GET", "/workflows", nil, &out)
	return out.Data, err
}

func (c *Client) GetDefinition(ctx context.Context, id string) (*Definition, error) {
	var out struct {
		Data struct {
			Definition Definition `json:"definition"`
		} `json:"data"`
	}
	err := c.do(ctx, "GET", fmt.Sprintf("/workflows/%s/definition", id), nil, &out)
	return &out.Data.Definition, err
}

func (c *Client) GetVariables(ctx context.Context, workflowID string) ([]Variable, error) {
	var out struct {
		Data struct {
			Fixed []Variable `json:"fixed"`
		} `json:"data"`
	}
	err := c.do(ctx, "GET", fmt.Sprintf("/workflows/%s/variables", workflowID), nil, &out)
	return out.Data.Fixed, err
}

func (c *Client) StartExecution(ctx context.Context, workflowID, phoneNumber string, variables map[string]interface{}) (*ExecutionResponse, error) {
	body := map[string]interface{}{
		"workflow_execution": map[string]interface{}{
			"phone_number": phoneNumber,
			"variables":    variables,
		},
	}

	var out struct {
		Data ExecutionResponse `json:"data"`
	}
	err := c.do(ctx, "POST", fmt.Sprintf("/workflows/%s/executions", workflowID), body, &out)
	if err != nil {
		return nil, err
	}
	return &out.Data, nil
}

func (c *Client) GetExecution(ctx context.Context, workflowID, executionID string) (*ExecutionStatus, error) {
	var out struct {
		Data ExecutionStatus `json:"data"`
	}
	err := c.do(ctx, "GET", fmt.Sprintf("/workflow_executions/%s", executionID), nil, &out)
	if err != nil {
		return nil, err
	}
	return &out.Data, nil
}

func (c *Client) ResumeExecution(ctx context.Context, workflowID, executionID, data string) error {
	body := map[string]interface{}{
		"message": map[string]interface{}{
			"kind": "payload",
			"data": data,
		},
	}
	return c.do(ctx, "POST", fmt.Sprintf("/workflow_executions/%s/resume", executionID), body, nil)
}

func (c *Client) UpdateExecutionStatus(ctx context.Context, workflowID, executionID, status string) error {
	body := map[string]interface{}{
		"workflow_execution": map[string]interface{}{
			"status": status,
		},
	}
	return c.do(ctx, "PATCH", fmt.Sprintf("/workflow_executions/%s", executionID), body, nil)
}

func (c *Client) GetEvents(ctx context.Context, workflowID, executionID string) ([]Event, error) {
	var out struct {
		Data []Event `json:"data"`
	}
	err := c.do(ctx, "GET", fmt.Sprintf("/workflow_executions/%s/events", executionID), nil, &out)
	return out.Data, err
}

func (c *Client) GetTriggers(ctx context.Context, workflowID string) ([]Trigger, error) {
	var out struct {
		Data []Trigger `json:"data"`
	}
	err := c.do(ctx, "GET", fmt.Sprintf("/workflows/%s/triggers", workflowID), nil, &out)
	return out.Data, err
}
