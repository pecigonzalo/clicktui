// Package clickup — shared domain models and error types.
package clickup

import "fmt"

// APIError represents a non-2xx response from the ClickUp API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("clickup api error %d: %s", e.StatusCode, e.Body)
}

// User represents a ClickUp user (abbreviated).
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// Team represents a ClickUp workspace (called "team" in the v2 API).
type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Space represents a ClickUp space within a workspace.
type Space struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Folder represents a ClickUp folder within a space.
type Folder struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Lists []List `json:"lists"`
}

// List represents a ClickUp list (inside a folder or folderless).
type List struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Statuses []Status `json:"statuses,omitempty"`
}

// Status represents a task status in ClickUp.
type Status struct {
	Status string `json:"status"`
	Color  string `json:"color"`
	Type   string `json:"type"`
}

// Assignee represents a user assigned to a task.
type Assignee struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// Task represents a ClickUp task with key metadata.
type Task struct {
	ID          string     `json:"id"`
	CustomID    string     `json:"custom_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Status      Status     `json:"status"`
	Priority    *Priority  `json:"priority"`
	Assignees   []Assignee `json:"assignees"`
	Tags        []Tag      `json:"tags"`
	DueDate     string     `json:"due_date"`
	StartDate   string     `json:"start_date"`
	DateCreated string     `json:"date_created"`
	DateUpdated string     `json:"date_updated"`
	URL         string     `json:"url"`
	Parent      string     `json:"parent"`
	Subtasks    []Task     `json:"subtasks,omitempty"`
	List        TaskRef    `json:"list"`
	Folder      TaskRef    `json:"folder"`
	Space       TaskRef    `json:"space"`
}

// Priority represents a task's priority level.
type Priority struct {
	ID    string `json:"id"`
	Color string `json:"color"`
	Name  string `json:"priority"`
}

// Tag represents a ClickUp tag.
type Tag struct {
	Name string `json:"name"`
}

// TaskRef is a lightweight reference to a list/folder/space on a task.
type TaskRef struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}
