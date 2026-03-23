// Package clickup — task endpoints.
package clickup

import (
	"context"
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
