// Package clickup — task endpoints.
package clickup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type tasksResponse struct {
	Tasks []Task `json:"tasks"`
}

// Tasks returns tasks for the given list. Page is zero-indexed.
func (c *Client) Tasks(ctx context.Context, listID string, page int) ([]Task, error) {
	var out tasksResponse
	path := fmt.Sprintf("/list/%s/task?page=%d&subtasks=true", listID, page)
	if err := c.do(ctx, "GET", path, &out); err != nil {
		return nil, err
	}
	return out.Tasks, nil
}

// Task returns a single task by ID with full details.
func (c *Client) Task(ctx context.Context, taskID string) (*Task, error) {
	var out Task
	if err := c.do(ctx, "GET", fmt.Sprintf("/task/%s?include_subtasks=true", taskID), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListStatuses returns the available statuses for a list.
// Statuses are workspace- and list-specific; never hard-code them.
func (c *Client) ListStatuses(ctx context.Context, listID string) ([]Status, error) {
	var out List
	if err := c.do(ctx, "GET", fmt.Sprintf("/list/%s", listID), &out); err != nil {
		return nil, err
	}
	return out.Statuses, nil
}

type updateTaskStatusRequest struct {
	Status string `json:"status"`
}

// UpdateTaskStatus sets the status of a task to the given value.
// The status string must be a valid status name returned by ListStatuses.
func (c *Client) UpdateTaskStatus(ctx context.Context, taskID, status string) (*Task, error) {
	body, err := json.Marshal(updateTaskStatusRequest{Status: status})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	var out Task
	if err := c.doWithBody(ctx, "PUT", fmt.Sprintf("/task/%s", taskID), bytes.NewReader(body), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// MoveTaskToList moves a task's home list using the dedicated v3 endpoint.
func (c *Client) MoveTaskToList(ctx context.Context, workspaceID, taskID, listID string) (*Task, error) {
	// This operation is available only in v3, while the rest of this client
	// targets v2. Use an absolute URL request and shared auth/HTTP plumbing.
	v3Base := strings.TrimSuffix(c.baseURL, "/api/v2")
	url := fmt.Sprintf("%s/api/v3/workspaces/%s/tasks/%s/home_list/%s", v3Base, workspaceID, taskID, listID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if err := c.provider.Authorize(ctx, req); err != nil {
		return nil, fmt.Errorf("authorize request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http PUT v3 move task: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}
	var out Task
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}

// UpdateTask updates the fields of an existing task specified by taskID.
// Only the fields set in req are sent; omitted fields remain unchanged.
func (c *Client) UpdateTask(ctx context.Context, taskID string, req UpdateTaskRequest) (*Task, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	var out Task
	if err := c.doWithBody(ctx, "PUT", fmt.Sprintf("/task/%s", taskID), bytes.NewReader(body), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateTask creates a new task in the given list and returns the created task.
func (c *Client) CreateTask(ctx context.Context, listID string, req CreateTaskRequest) (*Task, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	var out Task
	if err := c.doWithBody(ctx, "POST", fmt.Sprintf("/list/%s/task", listID), bytes.NewReader(body), &out); err != nil {
		return nil, err
	}
	return &out, nil
}
