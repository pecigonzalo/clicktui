// Package app -- pure sort engine for task lists.
//
// SortTasks reorders task summaries while preserving parent/subtask adjacency:
// parent-level groups are sorted, subtasks stay immediately below their parent.
package app

import (
	"cmp"
	"slices"
	"strings"
)

// SortField enumerates the supported task sort fields.
const (
	SortFieldStatus   = "status"
	SortFieldPriority = "priority"
	SortFieldDueDate  = "due_date"
	SortFieldAssignee = "assignee"
	SortFieldName     = "name"
)

// ValidSortFields lists all supported sort field names in cycle order.
var ValidSortFields = []string{
	SortFieldStatus,
	SortFieldPriority,
	SortFieldDueDate,
	SortFieldAssignee,
	SortFieldName,
}

// SortTasks returns a copy of tasks sorted by the given field and direction.
// statusOrder maps lowercased status name to its position (0-indexed) in the
// list's workflow; pass nil when unavailable (falls back to alphabetical).
//
// Subtask adjacency is preserved: parent-level groups are sorted as units,
// and subtasks remain immediately below their parent in their original order.
//
// Empty or missing values for the sort key always sort last regardless of
// direction.
func SortTasks(tasks []TaskSummary, field string, ascending bool, statusOrder map[string]int) []TaskSummary {
	if len(tasks) <= 1 || field == "" {
		result := make([]TaskSummary, len(tasks))
		copy(result, tasks)
		return result
	}

	// Build parent groups: each group is a parent task followed by its children.
	groups := buildGroups(tasks)

	// Sort groups by the parent's sort key.
	sortKey := keyFunc(field, statusOrder)
	slices.SortStableFunc(groups, func(a, b taskGroup) int {
		ka := sortKey(a.parent)
		kb := sortKey(b.parent)
		// Empty values always sort last, regardless of direction.
		if ka.empty != kb.empty {
			if ka.empty {
				return 1
			}
			return -1
		}
		c := compareKeys(ka, kb)
		if !ascending {
			c = -c
		}
		return c
	})

	// Flatten groups back into a single slice.
	result := make([]TaskSummary, 0, len(tasks))
	for _, g := range groups {
		result = append(result, g.parent)
		result = append(result, g.children...)
	}
	return result
}

// taskGroup represents a parent task and its immediate children.
type taskGroup struct {
	parent   TaskSummary
	children []TaskSummary
}

// buildGroups partitions tasks into parent groups.
// Tasks with a Parent field pointing to another task in the slice are grouped
// under that parent. Orphan subtasks (parent not in slice) are treated as
// top-level.
func buildGroups(tasks []TaskSummary) []taskGroup {
	// Build set of IDs present.
	present := make(map[string]struct{}, len(tasks))
	for _, t := range tasks {
		present[t.ID] = struct{}{}
	}

	// Map parentID -> children in order of appearance.
	childrenOf := make(map[string][]TaskSummary)
	for _, t := range tasks {
		if t.Parent != "" {
			if _, ok := present[t.Parent]; ok {
				childrenOf[t.Parent] = append(childrenOf[t.Parent], t)
			}
		}
	}

	// Walk tasks, emitting groups for top-level/orphan tasks.
	placed := make(map[string]struct{}, len(tasks))
	var groups []taskGroup

	for _, t := range tasks {
		if _, ok := placed[t.ID]; ok {
			continue
		}
		// Skip if this is a child of a present parent (will be grouped under parent).
		if t.Parent != "" {
			if _, ok := present[t.Parent]; ok {
				continue
			}
		}
		placed[t.ID] = struct{}{}
		g := taskGroup{parent: t}
		for _, child := range childrenOf[t.ID] {
			placed[child.ID] = struct{}{}
			g.children = append(g.children, child)
		}
		groups = append(groups, g)
	}

	return groups
}

// sortKeyValue wraps a comparable key with an empty flag.
// Empty keys always sort last.
type sortKeyValue struct {
	intKey int
	strKey string
	empty  bool
}

// keyFunc returns a function that extracts a sortKeyValue from a TaskSummary
// for the given field.
func keyFunc(field string, statusOrder map[string]int) func(TaskSummary) sortKeyValue {
	switch field {
	case SortFieldStatus:
		return func(t TaskSummary) sortKeyValue {
			if t.Status == "" {
				return sortKeyValue{empty: true}
			}
			lower := strings.ToLower(t.Status)
			if statusOrder != nil {
				if idx, ok := statusOrder[lower]; ok {
					return sortKeyValue{intKey: idx}
				}
			}
			// Fallback: alphabetical.
			return sortKeyValue{strKey: lower}
		}
	case SortFieldPriority:
		return func(t TaskSummary) sortKeyValue {
			if t.PriorityOrder == 5 { // none
				return sortKeyValue{empty: true}
			}
			return sortKeyValue{intKey: t.PriorityOrder}
		}
	case SortFieldDueDate:
		return func(t TaskSummary) sortKeyValue {
			if t.DueDate == "" {
				return sortKeyValue{empty: true}
			}
			return sortKeyValue{strKey: t.DueDate}
		}
	case SortFieldAssignee:
		return func(t TaskSummary) sortKeyValue {
			if t.Assignee == "" {
				return sortKeyValue{empty: true}
			}
			return sortKeyValue{strKey: strings.ToLower(t.Assignee)}
		}
	case SortFieldName:
		return func(t TaskSummary) sortKeyValue {
			if t.Name == "" {
				return sortKeyValue{empty: true}
			}
			return sortKeyValue{strKey: strings.ToLower(t.Name)}
		}
	default:
		return func(_ TaskSummary) sortKeyValue { return sortKeyValue{} }
	}
}

// compareKeys compares two sort key values.
// Empty keys always sort after non-empty keys.
func compareKeys(a, b sortKeyValue) int {
	// Empty values always sort last.
	if a.empty && b.empty {
		return 0
	}
	if a.empty {
		return 1
	}
	if b.empty {
		return -1
	}

	// Integer keys take precedence when both are set.
	if a.intKey != 0 || b.intKey != 0 {
		return cmp.Compare(a.intKey, b.intKey)
	}

	return cmp.Compare(a.strKey, b.strKey)
}

// NextSortField returns the next sort field in the cycle after current.
// If current is empty or not found, returns the first field.
// If current is the last field, returns "" (no sort).
func NextSortField(current string) string {
	if current == "" {
		return ValidSortFields[0]
	}
	for i, f := range ValidSortFields {
		if f == current {
			if i+1 < len(ValidSortFields) {
				return ValidSortFields[i+1]
			}
			return "" // cycle back to no sort
		}
	}
	return ValidSortFields[0]
}
