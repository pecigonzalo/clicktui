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
	Parent   string // parent task ID, empty for top-level tasks
}

// SubtaskSummary is a lightweight summary of a subtask for the detail view.
type SubtaskSummary struct {
	ID     string
	Name   string
	Status string
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
	ListID      string
	Folder      string
	Space       string
	Subtasks    []SubtaskSummary
}

// StatusOption is a display-oriented status value for the status picker.
type StatusOption struct {
	Name  string
	Color string
	Type  string
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
// Summaries are ordered so that each parent task is immediately followed by
// its children, preserving original API order among peers.
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
			Parent:   t.Parent,
		}
	}
	return orderByParent(summaries), nil
}

// LoadTaskDetail returns full details for a single task.
func (s *TaskService) LoadTaskDetail(ctx context.Context, taskID string) (*TaskDetail, error) {
	t, err := s.api.Task(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("load task detail: %w", err)
	}
	return taskToDetail(t), nil
}

// LoadListStatuses returns the available statuses for a list.
// Statuses are list-specific and sourced live from the ClickUp API.
func (s *TaskService) LoadListStatuses(ctx context.Context, listID string) ([]StatusOption, error) {
	statuses, err := s.api.ListStatuses(ctx, listID)
	if err != nil {
		return nil, fmt.Errorf("load list statuses: %w", err)
	}
	opts := make([]StatusOption, len(statuses))
	for i, st := range statuses {
		opts[i] = StatusOption{
			Name:  st.Status,
			Color: st.Color,
			Type:  st.Type,
		}
	}
	return opts, nil
}

// UpdateTaskStatus sets a task's status to the given value and returns the
// refreshed task detail.  The status string must be a live value obtained via
// LoadListStatuses; no status values are hard-coded here.
func (s *TaskService) UpdateTaskStatus(ctx context.Context, taskID, status string) (*TaskDetail, error) {
	t, err := s.api.UpdateTaskStatus(ctx, taskID, status)
	if err != nil {
		return nil, fmt.Errorf("update task status: %w", err)
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
	subtasks := make([]SubtaskSummary, len(t.Subtasks))
	for i, st := range t.Subtasks {
		subtasks[i] = SubtaskSummary{
			ID:     st.ID,
			Name:   st.Name,
			Status: st.Status.Status,
		}
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
		ListID:      t.List.ID,
		Folder:      t.Folder.Name,
		Space:       t.Space.Name,
		Subtasks:    subtasks,
	}
}

// orderByParent reorders a flat list of task summaries so that each parent is
// immediately followed by its children. The relative order among top-level
// tasks and among siblings is preserved from the original input.
// Orphan subtasks (whose parent is not in the slice) are treated as top-level.
// The input slice is not mutated.
func orderByParent(tasks []TaskSummary) []TaskSummary {
	if len(tasks) == 0 {
		return tasks
	}

	// Build a set of IDs present in the input.
	present := make(map[string]struct{}, len(tasks))
	for _, t := range tasks {
		present[t.ID] = struct{}{}
	}

	// Build parentID → children map, preserving input order.
	children := make(map[string][]TaskSummary)
	for _, t := range tasks {
		if t.Parent != "" {
			if _, ok := present[t.Parent]; ok {
				children[t.Parent] = append(children[t.Parent], t)
			}
		}
	}

	result := make([]TaskSummary, 0, len(tasks))
	placed := make(map[string]struct{}, len(tasks))

	// Walk the input: emit top-level tasks and orphans, followed by children.
	for _, t := range tasks {
		if _, ok := placed[t.ID]; ok {
			continue
		}
		isChildOfPresent := false
		if t.Parent != "" {
			if _, ok := present[t.Parent]; ok {
				isChildOfPresent = true
			}
		}
		if isChildOfPresent {
			// Will be emitted after its parent.
			continue
		}
		result = append(result, t)
		placed[t.ID] = struct{}{}
		for _, child := range children[t.ID] {
			result = append(result, child)
			placed[child.ID] = struct{}{}
		}
	}

	return result
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
