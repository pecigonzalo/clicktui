// Package app — task service for loading task lists and details.
package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/pecigonzalo/clicktui/internal/clickup"
)

// TaskSummary is a display-oriented view of a task for the list pane.
type TaskSummary struct {
	ID          string
	Name        string
	Status      string
	StatusColor string // hex color from the API, e.g. "#ff6b6b"; empty when absent
	Priority    string
	Parent      string // parent task ID, empty for top-level tasks
	// Sort-friendly fields below; not rendered directly but used for ordering.
	DueDate       string // ISO date (YYYY-MM-DD) or empty; sortable as string
	Assignee      string // first assignee username or empty
	PriorityOrder int    // 1=urgent, 2=high, 3=normal, 4=low, 0=none (sorts last)
}

// SubtaskSummary is a lightweight summary of a subtask for the detail view.
type SubtaskSummary struct {
	ID          string
	Name        string
	Status      string
	StatusColor string // hex color from the API, e.g. "#ff6b6b"; empty when absent
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
	AssigneeIDs []int // user IDs corresponding to Assignees, for mutation use
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

// MemberSummary is a display-oriented view of a list member.
type MemberSummary struct {
	ID       int
	Username string
	Email    string
}

// UpdateTaskInput carries the fields that may be mutated on an existing task.
// Pointer fields are optional; nil means "leave unchanged".
type UpdateTaskInput struct {
	Name         *string
	Description  *string
	DueDate      *string // epoch ms string, or empty string to clear
	Priority     *int
	AssigneesAdd []int
	AssigneesRem []int
	Status       *string
}

// CreateTaskInput carries the fields for creating a new task in a list.
type CreateTaskInput struct {
	Name        string
	Status      string
	Description string
	DueDate     string // epoch ms string
	Priority    int
	Assignees   []int
}

// TaskService loads and transforms task data for presentation.
// It caches results in memory to reduce API calls; caches are invalidated
// on mutations and can be explicitly evicted for manual refresh.
//
// A singleflight.Group deduplicates concurrent cache-miss fetches for the
// same key.  Invalidation methods call group.Forget so the next request
// after eviction always reaches the API.
type TaskService struct {
	api ClickUpAPI

	mu              sync.Mutex
	taskDetailCache map[string]*TaskDetail     // keyed by taskID
	taskListCache   map[string][]TaskSummary   // keyed by "listID:page"
	statusCache     map[string][]StatusOption  // keyed by listID
	memberCache     map[string][]MemberSummary // keyed by listID

	group singleflight.Group
}

// NewTaskService creates a TaskService backed by the given API.
func NewTaskService(api ClickUpAPI) *TaskService {
	return &TaskService{
		api:             api,
		taskDetailCache: make(map[string]*TaskDetail),
		taskListCache:   make(map[string][]TaskSummary),
		statusCache:     make(map[string][]StatusOption),
		memberCache:     make(map[string][]MemberSummary),
	}
}

// LoadTasks returns a page of task summaries for a list.
// Results are cached by listID and page; subsequent calls for the same listID
// and page return the cached value without an API call.
// Use InvalidateTaskList to force a refresh of all pages for a list.
// Summaries are ordered so that each parent task is immediately followed by
// its children, preserving original API order among peers.
func (s *TaskService) LoadTasks(ctx context.Context, listID string, page int) ([]TaskSummary, error) {
	cacheKey := fmt.Sprintf("%s:%d", listID, page)
	sfKey := "tasks:" + cacheKey

	s.mu.Lock()
	if cached, ok := s.taskListCache[cacheKey]; ok {
		s.mu.Unlock()
		return cached, nil
	}
	s.mu.Unlock()

	v, err, _ := s.group.Do(sfKey, func() (any, error) {
		tasks, err := s.api.Tasks(ctx, listID, page)
		if err != nil {
			return nil, fmt.Errorf("load tasks: %w", err)
		}
		summaries := make([]TaskSummary, len(tasks))
		for i, t := range tasks {
			summaries[i] = TaskSummary{
				ID:            t.ID,
				Name:          t.Name,
				Status:        t.Status.Status,
				StatusColor:   t.Status.Color,
				Priority:      priorityName(t.Priority),
				Parent:        t.Parent,
				DueDate:       formatEpochMillis(t.DueDate),
				Assignee:      firstAssignee(t.Assignees),
				PriorityOrder: priorityOrder(t.Priority),
			}
		}
		result := orderByParent(summaries)

		s.mu.Lock()
		s.taskListCache[cacheKey] = result
		s.mu.Unlock()

		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]TaskSummary), nil
}

// LoadTaskDetail returns full details for a single task.
// Results are cached by taskID; subsequent calls return the cached value.
// Use InvalidateTaskDetail to force a refresh.
func (s *TaskService) LoadTaskDetail(ctx context.Context, taskID string) (*TaskDetail, error) {
	sfKey := "detail:" + taskID

	s.mu.Lock()
	if cached, ok := s.taskDetailCache[taskID]; ok {
		s.mu.Unlock()
		return cached, nil
	}
	s.mu.Unlock()

	v, err, _ := s.group.Do(sfKey, func() (any, error) {
		t, err := s.api.Task(ctx, taskID)
		if err != nil {
			return nil, fmt.Errorf("load task detail: %w", err)
		}
		detail := taskToDetail(t)

		s.mu.Lock()
		s.taskDetailCache[taskID] = detail
		s.mu.Unlock()

		return detail, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*TaskDetail), nil
}

// LoadListStatuses returns the available statuses for a list.
// Results are cached by listID; subsequent calls return the cached value.
func (s *TaskService) LoadListStatuses(ctx context.Context, listID string) ([]StatusOption, error) {
	sfKey := "statuses:" + listID

	s.mu.Lock()
	if cached, ok := s.statusCache[listID]; ok {
		s.mu.Unlock()
		return cached, nil
	}
	s.mu.Unlock()

	v, err, _ := s.group.Do(sfKey, func() (any, error) {
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

		s.mu.Lock()
		s.statusCache[listID] = opts
		s.mu.Unlock()

		return opts, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]StatusOption), nil
}

// UpdateTaskStatus sets a task's status to the given value and returns the
// refreshed task detail.  The status string must be a live value obtained via
// LoadListStatuses; no status values are hard-coded here.
//
// On success the task detail cache for taskID is replaced with the fresh
// result and all task list cache entries are evicted (since we don't track
// which list contains the task).  In-flight singleflight requests for the
// affected keys are also forgotten so subsequent loads hit the API.
func (s *TaskService) UpdateTaskStatus(ctx context.Context, taskID, status string) (*TaskDetail, error) {
	t, err := s.api.UpdateTaskStatus(ctx, taskID, status)
	if err != nil {
		return nil, fmt.Errorf("update task status: %w", err)
	}
	detail := taskToDetail(t)

	s.mu.Lock()
	// Collect list-cache keys before clearing so we can forget their
	// singleflight entries outside the lock.
	listKeys := make([]string, 0, len(s.taskListCache))
	for key := range s.taskListCache {
		listKeys = append(listKeys, key)
	}
	delete(s.taskDetailCache, taskID)
	clear(s.taskListCache)
	s.taskDetailCache[taskID] = detail
	s.mu.Unlock()

	// Forget singleflight keys so any in-flight fetch is discarded.
	s.group.Forget("detail:" + taskID)
	for _, key := range listKeys {
		s.group.Forget("tasks:" + key)
	}

	return detail, nil
}

// MoveTaskToList moves a task to another list and returns refreshed task detail.
//
// On success the task detail cache for taskID is replaced with the fresh
// result, all task list caches are evicted, and status caches for the old/new
// lists are evicted (status values are list-specific).
func (s *TaskService) MoveTaskToList(ctx context.Context, workspaceID, taskID, listID string) (*TaskDetail, error) {
	moved, err := s.api.MoveTaskToList(ctx, workspaceID, taskID, listID)
	if err != nil {
		return nil, fmt.Errorf("move task to list: %w", err)
	}

	// The move endpoint can return a sparse task payload. Fetch full detail so
	// the detail pane remains fully populated after the move.
	fresh, err := s.api.Task(ctx, taskID)
	if err != nil {
		// Best effort fallback to the move response if immediate refetch fails.
		fresh = moved
	}
	detail := taskToDetail(fresh)
	// Some move responses omit nested list fields. Keep destination list context
	// so follow-up list reloads do not attempt an empty list ID.
	if detail.ListID == "" {
		detail.ListID = listID
	}
	if detail.List == "" {
		detail.List = listID
	}

	oldListID := ""
	newListID := detail.ListID
	s.mu.Lock()
	if cached, ok := s.taskDetailCache[taskID]; ok {
		oldListID = cached.ListID
	}
	// Collect list-cache keys before clearing so we can forget their
	// singleflight entries outside the lock.
	listKeys := make([]string, 0, len(s.taskListCache))
	for key := range s.taskListCache {
		listKeys = append(listKeys, key)
	}
	delete(s.taskDetailCache, taskID)
	clear(s.taskListCache)
	if oldListID != "" {
		delete(s.statusCache, oldListID)
	}
	if newListID != "" {
		delete(s.statusCache, newListID)
	}
	s.taskDetailCache[taskID] = detail
	s.mu.Unlock()

	s.group.Forget("detail:" + taskID)
	for _, key := range listKeys {
		s.group.Forget("tasks:" + key)
	}
	if oldListID != "" {
		s.group.Forget("statuses:" + oldListID)
	}
	if newListID != "" {
		s.group.Forget("statuses:" + newListID)
	}

	return detail, nil
}

// InvalidateTaskList evicts all cached task list pages for listID so the next
// LoadTasks call fetches fresh data from the API.  Any in-flight singleflight
// requests for those pages are also forgotten.
func (s *TaskService) InvalidateTaskList(listID string) {
	prefix := listID + ":"
	var forgotKeys []string
	s.mu.Lock()
	for key := range s.taskListCache {
		if strings.HasPrefix(key, prefix) {
			delete(s.taskListCache, key)
			forgotKeys = append(forgotKeys, key)
		}
	}
	s.mu.Unlock()

	for _, key := range forgotKeys {
		s.group.Forget("tasks:" + key)
	}
}

// InvalidateTaskDetail evicts the cached task detail for taskID so the next
// LoadTaskDetail call fetches fresh data from the API.  Any in-flight
// singleflight request for this taskID is also forgotten.
func (s *TaskService) InvalidateTaskDetail(taskID string) {
	s.mu.Lock()
	delete(s.taskDetailCache, taskID)
	s.mu.Unlock()

	s.group.Forget("detail:" + taskID)
}

// UpdateTask updates the writable fields of an existing task and returns an
// error on failure.  On success the task detail cache for taskID is evicted
// and all task list cache entries are invalidated, forcing fresh loads.
func (s *TaskService) UpdateTask(ctx context.Context, taskID string, input UpdateTaskInput) error {
	req := clickup.UpdateTaskRequest{
		Name:        input.Name,
		Description: input.Description,
		DueDate:     input.DueDate,
		Priority:    input.Priority,
		Status:      input.Status,
	}
	if len(input.AssigneesAdd) > 0 || len(input.AssigneesRem) > 0 {
		req.Assignees = &clickup.AssigneeUpdate{
			Add: input.AssigneesAdd,
			Rem: input.AssigneesRem,
		}
	}

	t, err := s.api.UpdateTask(ctx, taskID, req)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	detail := taskToDetail(t)

	s.mu.Lock()
	listKeys := make([]string, 0, len(s.taskListCache))
	for key := range s.taskListCache {
		listKeys = append(listKeys, key)
	}
	delete(s.taskDetailCache, taskID)
	clear(s.taskListCache)
	s.taskDetailCache[taskID] = detail
	s.mu.Unlock()

	s.group.Forget("detail:" + taskID)
	for _, key := range listKeys {
		s.group.Forget("tasks:" + key)
	}

	return nil
}

// CreateTask creates a new task in the given list and returns the new task's ID.
// On success all task list cache entries for listID are invalidated.
func (s *TaskService) CreateTask(ctx context.Context, listID string, input CreateTaskInput) (string, error) {
	req := clickup.CreateTaskRequest{
		Name:        input.Name,
		Status:      input.Status,
		Description: input.Description,
		DueDate:     input.DueDate,
		Priority:    input.Priority,
		Assignees:   input.Assignees,
	}

	t, err := s.api.CreateTask(ctx, listID, req)
	if err != nil {
		return "", fmt.Errorf("create task: %w", err)
	}

	s.InvalidateTaskList(listID)

	return t.ID, nil
}

// LoadMembers returns the members of a list.
// Results are cached by listID; subsequent calls return the cached value.
func (s *TaskService) LoadMembers(ctx context.Context, listID string) ([]MemberSummary, error) {
	sfKey := "members:" + listID

	s.mu.Lock()
	if cached, ok := s.memberCache[listID]; ok {
		s.mu.Unlock()
		return cached, nil
	}
	s.mu.Unlock()

	v, err, _ := s.group.Do(sfKey, func() (any, error) {
		members, err := s.api.ListMembers(ctx, listID)
		if err != nil {
			return nil, fmt.Errorf("load members: %w", err)
		}
		summaries := make([]MemberSummary, len(members))
		for i, m := range members {
			summaries[i] = MemberSummary{
				ID:       m.ID,
				Username: m.Username,
				Email:    m.Email,
			}
		}

		s.mu.Lock()
		s.memberCache[listID] = summaries
		s.mu.Unlock()

		return summaries, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]MemberSummary), nil
}

func taskToDetail(t *clickup.Task) *TaskDetail {
	assignees := make([]string, len(t.Assignees))
	assigneeIDs := make([]int, len(t.Assignees))
	for i, a := range t.Assignees {
		assignees[i] = a.Username
		assigneeIDs[i] = a.ID
	}
	tags := make([]string, len(t.Tags))
	for i, tag := range t.Tags {
		tags[i] = tag.Name
	}
	subtasks := make([]SubtaskSummary, len(t.Subtasks))
	for i, st := range t.Subtasks {
		subtasks[i] = SubtaskSummary{
			ID:          st.ID,
			Name:        st.Name,
			Status:      st.Status.Status,
			StatusColor: st.Status.Color,
		}
	}
	return &TaskDetail{
		ID:          t.ID,
		CustomID:    t.CustomID,
		Name:        t.Name,
		Description: strings.TrimSpace(t.Description),
		Status:      t.Status.Status,
		StatusColor: t.Status.Color,
		Priority:    priorityName(t.Priority),
		Assignees:   assignees,
		AssigneeIDs: assigneeIDs,
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

// priorityOrder maps a ClickUp priority to a sort-friendly integer.
// ClickUp uses 1=urgent, 2=high, 3=normal, 4=low; nil means none.
// We keep 1-4 as-is and map none to 5 so it sorts last.
func priorityOrder(p *clickup.Priority) int {
	if p == nil || p.ID == "" || p.ID == "0" {
		return 5 // none sorts last
	}
	switch p.ID {
	case "1":
		return 1 // urgent
	case "2":
		return 2 // high
	case "3":
		return 3 // normal
	case "4":
		return 4 // low
	default:
		return 5
	}
}

// firstAssignee returns the username of the first assignee, or "" when
// there are no assignees.
func firstAssignee(assignees []clickup.Assignee) string {
	if len(assignees) == 0 {
		return ""
	}
	return assignees[0].Username
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
