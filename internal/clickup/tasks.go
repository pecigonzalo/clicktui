// Package clickup — task endpoints.
package clickup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

type tasksResponse struct {
	Tasks []Task `json:"tasks"`
}

// Tasks returns tasks for the given list. Page is zero-indexed.
func (c *Client) Tasks(ctx context.Context, listID string, page int) ([]Task, error) {
	var out tasksResponse
	path := fmt.Sprintf("/list/%s/task?page=%d", listID, page)
	if err := c.do(ctx, "GET", path, &out); err != nil {
		return nil, err
	}
	return out.Tasks, nil
}

// Task returns a single task by ID with full details.
func (c *Client) Task(ctx context.Context, taskID string) (*Task, error) {
	var out Task
	if err := c.do(ctx, "GET", fmt.Sprintf("/task/%s", taskID), &out); err != nil {
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
