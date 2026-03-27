package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// ── ParseTaskQuery ──────────────────────────────────────────────────────────

func TestParseTaskQuery(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      []app.FieldFilter
		wantFree  string
		wantEmpty bool
	}{
		{
			name:     "status only",
			input:    "status:todo",
			want:     []app.FieldFilter{{Field: "status", Value: "todo"}},
			wantFree: "",
		},
		{
			name:     "priority only",
			input:    "priority:high",
			want:     []app.FieldFilter{{Field: "priority", Value: "high"}},
			wantFree: "",
		},
		{
			name:     "multi-word value",
			input:    "status:in progress",
			want:     []app.FieldFilter{{Field: "status", Value: "in progress"}},
			wantFree: "",
		},
		{
			name:     "field greedy value",
			input:    "status:todo auth",
			want:     []app.FieldFilter{{Field: "status", Value: "todo auth"}},
			wantFree: "",
		},
		{
			name:     "free text before field",
			input:    "auth status:todo",
			want:     []app.FieldFilter{{Field: "status", Value: "todo"}},
			wantFree: "auth",
		},
		{
			name:  "multiple fields",
			input: "status:todo priority:high",
			want: []app.FieldFilter{
				{Field: "status", Value: "todo"},
				{Field: "priority", Value: "high"},
			},
			wantFree: "",
		},
		{
			name:      "empty string",
			input:     "",
			want:      nil,
			wantFree:  "",
			wantEmpty: true,
		},
		{
			name:      "whitespace only",
			input:     "   ",
			want:      nil,
			wantFree:  "",
			wantEmpty: true,
		},
		{
			name:     "free text only",
			input:    "auth login",
			want:     nil,
			wantFree: "auth login",
		},
		{
			name:     "unknown field",
			input:    "foo:bar",
			want:     nil,
			wantFree: "foo:bar",
		},
		{
			name:      "partial field no value",
			input:     "status:",
			want:      nil,
			wantFree:  "",
			wantEmpty: true,
		},
		{
			name:     "case-insensitive field",
			input:    "Status:TODO",
			want:     []app.FieldFilter{{Field: "status", Value: "todo"}},
			wantFree: "",
		},
		{
			name:  "multi-word value followed by field",
			input: "status:in progress priority:high",
			want: []app.FieldFilter{
				{Field: "status", Value: "in progress"},
				{Field: "priority", Value: "high"},
			},
			wantFree: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			q := app.ParseTaskQuery(tt.input)
			assert.Equal(t, tt.want, q.Fields)
			assert.Equal(t, tt.wantFree, q.FreeText)
			if tt.wantEmpty {
				assert.True(t, q.Empty())
			} else {
				assert.Equal(t, len(tt.want) == 0 && tt.wantFree == "", q.Empty())
			}
		})
	}
}

// ── FilterTasks ─────────────────────────────────────────────────────────────

func makeTestTasks() []app.TaskSummary {
	return []app.TaskSummary{
		{ID: "1", Name: "Fix authentication bug", Status: "todo", Priority: "high"},
		{ID: "2", Name: "Update documentation", Status: "in progress", Priority: "normal"},
		{ID: "3", Name: "Refactor auth module", Status: "todo", Priority: "normal"},
		{ID: "4", Name: "Add login page", Status: "done", Priority: "low"},
		{ID: "5", Name: "Write unit tests", Status: "todo", Priority: "high"},
	}
}

func TestFilterTasks_EmptyQuery(t *testing.T) {
	result := app.FilterTasks(makeTestTasks(), app.TaskQuery{})
	assert.Nil(t, result, "empty query should return nil (show all)")
}

func TestFilterTasks_FieldMatchOnly(t *testing.T) {
	q := app.ParseTaskQuery("status:todo")
	result := app.FilterTasks(makeTestTasks(), q)
	require.Len(t, result, 3)
	for _, r := range result {
		assert.Equal(t, "todo", r.Status)
	}
}

func TestFilterTasks_FieldMatchCaseInsensitive(t *testing.T) {
	q := app.ParseTaskQuery("status:TODO")
	result := app.FilterTasks(makeTestTasks(), q)
	require.Len(t, result, 3)
}

func TestFilterTasks_FieldSubstringMatch(t *testing.T) {
	// "in progress" contains "progress"
	q := app.ParseTaskQuery("status:progress")
	result := app.FilterTasks(makeTestTasks(), q)
	require.Len(t, result, 1)
	assert.Equal(t, "2", result[0].ID)
}

func TestFilterTasks_FuzzyMatchOnly(t *testing.T) {
	q := app.ParseTaskQuery("auth")
	result := app.FilterTasks(makeTestTasks(), q)
	require.NotEmpty(t, result)
	// Should match "Fix authentication bug" and "Refactor auth module"
	ids := make([]string, len(result))
	for i, r := range result {
		ids[i] = r.ID
	}
	assert.Contains(t, ids, "1")
	assert.Contains(t, ids, "3")
}

func TestFilterTasks_FuzzyMatchOrdering(t *testing.T) {
	q := app.ParseTaskQuery("auth")
	result := app.FilterTasks(makeTestTasks(), q)
	require.NotEmpty(t, result)
	// "Refactor auth module" should score higher than "Fix authentication bug"
	// because "auth" is an exact substring in the name.
	assert.Equal(t, "3", result[0].ID, "exact substring match should rank first")
}

func TestFilterTasks_CombinedFieldAndFuzzy(t *testing.T) {
	// Free text before a field token: "auth status:todo"
	q := app.ParseTaskQuery("auth status:todo")
	result := app.FilterTasks(makeTestTasks(), q)
	require.NotEmpty(t, result)
	for _, r := range result {
		assert.Equal(t, "todo", r.Status)
	}
	// Should only include todo tasks that fuzzy-match "auth".
	ids := make([]string, len(result))
	for i, r := range result {
		ids[i] = r.ID
	}
	assert.Contains(t, ids, "1") // "Fix authentication bug" — status:todo
	assert.Contains(t, ids, "3") // "Refactor auth module" — status:todo
}

func TestFilterTasks_NoMatches(t *testing.T) {
	q := app.ParseTaskQuery("zzzzz")
	result := app.FilterTasks(makeTestTasks(), q)
	assert.Nil(t, result)
}

func TestFilterTasks_NoMatchesFieldFilter(t *testing.T) {
	q := app.ParseTaskQuery("status:cancelled")
	result := app.FilterTasks(makeTestTasks(), q)
	assert.Nil(t, result)
}

func TestFilterTasks_MultipleFieldFilters(t *testing.T) {
	q := app.ParseTaskQuery("status:todo priority:high")
	result := app.FilterTasks(makeTestTasks(), q)
	require.Len(t, result, 2)
	for _, r := range result {
		assert.Equal(t, "todo", r.Status)
		assert.Equal(t, "high", r.Priority)
	}
}

func TestFilterTasks_EmptyInput(t *testing.T) {
	result := app.FilterTasks(nil, app.ParseTaskQuery("auth"))
	assert.Nil(t, result)
}

func TestTaskQuery_Empty(t *testing.T) {
	assert.True(t, app.TaskQuery{}.Empty())
	assert.False(t, app.TaskQuery{FreeText: "x"}.Empty())
	assert.False(t, app.TaskQuery{Fields: []app.FieldFilter{{Field: "status", Value: "todo"}}}.Empty())
}
