// Package app — task service for loading task lists and details.
package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pecigonzalo/clicktui/internal/clickup"
)

// TaskSummary is a display-oriented view of a task for the list pane.
type TaskSummary struct {
	ID       string
	Name     string
	Status   string
	Priority string
}

// TaskDetail is a display-oriented view of a single task for the detail pane.
type TaskDetail struct {
	ID          string
	CustomID    string
	Name        string
	Description string
	Status      string
	StatusColor string
	Priority    string
	Assignees   []string
	Tags        []string
	DueDate     string
	StartDate   string
	DateCreated string
	DateUpdated string
	URL         string
	Parent      string
	List        string
	Folder      string
	Space       string
}

// TaskService loads and transforms task data for presentation.
type TaskService struct {
	api ClickUpAPI
}

// NewTaskService creates a TaskService backed by the given API.
func NewTaskService(api ClickUpAPI) *TaskService {
	return &TaskService{api: api}
}

// LoadTasks returns a page of task summaries for a list.
func (s *TaskService) LoadTasks(ctx context.Context, listID string, page int) ([]TaskSummary, error) {
	tasks, err := s.api.Tasks(ctx, listID, page)
	if err != nil {
		return nil, fmt.Errorf("load tasks: %w", err)
	}
	summaries := make([]TaskSummary, len(tasks))
	for i, t := range tasks {
		summaries[i] = TaskSummary{
			ID:       t.ID,
			Name:     t.Name,
			Status:   t.Status.Status,
			Priority: priorityName(t.Priority),
		}
	}
	return summaries, nil
}

// LoadTaskDetail returns full details for a single task.
func (s *TaskService) LoadTaskDetail(ctx context.Context, taskID string) (*TaskDetail, error) {
	t, err := s.api.Task(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("load task detail: %w", err)
	}
	return taskToDetail(t), nil
}

func taskToDetail(t *clickup.Task) *TaskDetail {
	assignees := make([]string, len(t.Assignees))
	for i, a := range t.Assignees {
		assignees[i] = a.Username
	}
	tags := make([]string, len(t.Tags))
	for i, tag := range t.Tags {
		tags[i] = tag.Name
	}
	return &TaskDetail{
		ID:          t.ID,
		CustomID:    t.CustomID,
		Name:        t.Name,
		Description: truncate(t.Description, 500),
		Status:      t.Status.Status,
		StatusColor: t.Status.Color,
		Priority:    priorityName(t.Priority),
		Assignees:   assignees,
		Tags:        tags,
		DueDate:     formatEpochMillis(t.DueDate),
		StartDate:   formatEpochMillis(t.StartDate),
		DateCreated: formatEpochMillis(t.DateCreated),
		DateUpdated: formatEpochMillis(t.DateUpdated),
		URL:         t.URL,
		Parent:      t.Parent,
		List:        t.List.Name,
		Folder:      t.Folder.Name,
		Space:       t.Space.Name,
	}
}

func priorityName(p *clickup.Priority) string {
	if p == nil {
		return "none"
	}
	return p.Name
}

func formatEpochMillis(s string) string {
	if s == "" || s == "null" {
		return ""
	}
	ms, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return s
	}
	return time.UnixMilli(ms).UTC().Format(time.DateOnly)
}

func truncate(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
