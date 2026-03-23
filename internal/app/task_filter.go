// Package app — pure filter engine for task lists.
//
// Parses a query string into structured field filters and free-text components,
// then applies them to task summaries using exact field matching and fuzzy name
// matching.
package app

import (
	"sort"
	"strings"

	"github.com/sahilm/fuzzy"
)

// knownFields maps recognised filter field names to TaskSummary accessor
// functions. Matching is case-insensitive on both the field name and value.
var knownFields = map[string]func(TaskSummary) string{
	"status":   func(t TaskSummary) string { return t.Status },
	"priority": func(t TaskSummary) string { return t.Priority },
}

// FieldFilter represents a single key:value filter predicate.
type FieldFilter struct {
	// Field is the normalised (lowercase) field name, e.g. "status".
	Field string
	// Value is the expected value, stored lowercase for case-insensitive matching.
	Value string
}

// TaskQuery is the parsed representation of a filter query string.
type TaskQuery struct {
	// Fields are key:value predicate filters (AND logic between them).
	Fields []FieldFilter
	// FreeText is the remaining query text used for fuzzy name matching.
	FreeText string
}

// Empty reports whether the query has no effective filter criteria.
func (q TaskQuery) Empty() bool {
	return len(q.Fields) == 0 && q.FreeText == ""
}

// ParseTaskQuery parses a filter input string into a structured TaskQuery.
//
// Field filters have the form "field:value" where value extends to the next
// field filter token or end of input. Multi-word values are supported
// (e.g. "status:in progress"). Anything that is not a field filter is
// collected as free text for fuzzy matching.
//
// Examples:
//
//	"status:todo"              → Fields=[{status,todo}], FreeText=""
//	"priority:high auth"       → Fields=[{priority,high}], FreeText="auth"
//	"status:in progress"       → Fields=[{status,in progress}], FreeText=""
//	"status:todo auth login"   → Fields=[{status,todo}], FreeText="auth login"
//	""                         → Empty query
func ParseTaskQuery(input string) TaskQuery {
	input = strings.TrimSpace(input)
	if input == "" {
		return TaskQuery{}
	}

	var q TaskQuery
	var freeWords []string

	// Tokenise by splitting on spaces, then greedily consume field:value runs.
	words := strings.Fields(input)
	i := 0
	for i < len(words) {
		word := words[i]
		colonIdx := strings.Index(word, ":")
		if colonIdx > 0 {
			field := strings.ToLower(word[:colonIdx])
			if _, ok := knownFields[field]; ok {
				// Consume the value: everything after the colon up to the next
				// known-field token or end of input.
				valuePart := word[colonIdx+1:]
				var valueParts []string
				if valuePart != "" {
					valueParts = append(valueParts, valuePart)
				}
				// Greedily consume subsequent words that are not field tokens.
				for i+1 < len(words) {
					next := words[i+1]
					if isFieldToken(next) {
						break
					}
					valueParts = append(valueParts, next)
					i++
				}
				value := strings.TrimSpace(strings.Join(valueParts, " "))
				if value != "" {
					q.Fields = append(q.Fields, FieldFilter{
						Field: field,
						Value: strings.ToLower(value),
					})
				}
				i++
				continue
			}
		}
		// Not a field filter — accumulate as free text.
		freeWords = append(freeWords, word)
		i++
	}

	q.FreeText = strings.TrimSpace(strings.Join(freeWords, " "))
	return q
}

// isFieldToken reports whether a word looks like a "field:..." token for a
// known field.
func isFieldToken(word string) bool {
	colonIdx := strings.Index(word, ":")
	if colonIdx <= 0 {
		return false
	}
	field := strings.ToLower(word[:colonIdx])
	_, ok := knownFields[field]
	return ok
}

// FilterTasks applies a TaskQuery to a slice of tasks and returns matching
// results. Field filters use case-insensitive substring matching. Free text
// uses fuzzy matching against the task Name. When free text is present,
// results are ordered by fuzzy score (best first). Both field filters AND
// free text must match (AND logic).
//
// Returns nil when no tasks match. Returns nil when the query is empty
// (callers should interpret nil as "show all").
func FilterTasks(tasks []TaskSummary, query TaskQuery) []TaskSummary {
	if query.Empty() {
		return nil
	}

	// First pass: apply field filters.
	candidates := tasks
	if len(query.Fields) > 0 {
		candidates = filterByFields(tasks, query.Fields)
	}

	if len(candidates) == 0 {
		return nil
	}

	// Second pass: apply fuzzy text matching on Name.
	if query.FreeText == "" {
		// Field-only query: preserve original order.
		result := make([]TaskSummary, len(candidates))
		copy(result, candidates)
		return result
	}

	return fuzzyMatchTasks(candidates, query.FreeText)
}

// filterByFields returns tasks where all field filters match (AND logic).
func filterByFields(tasks []TaskSummary, fields []FieldFilter) []TaskSummary {
	var result []TaskSummary
	for _, t := range tasks {
		if matchAllFields(t, fields) {
			result = append(result, t)
		}
	}
	return result
}

// matchAllFields reports whether a task matches every field filter.
func matchAllFields(t TaskSummary, fields []FieldFilter) bool {
	for _, f := range fields {
		accessor, ok := knownFields[f.Field]
		if !ok {
			return false
		}
		actual := strings.ToLower(accessor(t))
		if !strings.Contains(actual, f.Value) {
			return false
		}
	}
	return true
}

// taskNames adapts a TaskSummary slice for the fuzzy matching library.
type taskNames []TaskSummary

func (tn taskNames) String(i int) string { return tn[i].Name }
func (tn taskNames) Len() int            { return len(tn) }

// fuzzyMatchTasks returns tasks whose Name fuzzy-matches the pattern, ordered
// by match score (best first).
func fuzzyMatchTasks(tasks []TaskSummary, pattern string) []TaskSummary {
	matches := fuzzy.FindFrom(pattern, taskNames(tasks))
	if len(matches) == 0 {
		return nil
	}

	// Sort by score descending (higher = better match).
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	result := make([]TaskSummary, len(matches))
	for i, m := range matches {
		result[i] = tasks[m.Index]
	}
	return result
}
