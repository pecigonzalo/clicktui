package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// ── ParseTaskQuery ──────────────────────────────────────────────────────────

func TestParseTaskQuery_StatusOnly(t *testing.T) {
	q := app.ParseTaskQuery("status:todo")
	require.Len(t, q.Fields, 1)
	assert.Equal(t, "status", q.Fields[0].Field)
	assert.Equal(t, "todo", q.Fields[0].Value)
	assert.Empty(t, q.FreeText)
}

func TestParseTaskQuery_PriorityOnly(t *testing.T) {
	q := app.ParseTaskQuery("priority:high")
	require.Len(t, q.Fields, 1)
	assert.Equal(t, "priority", q.Fields[0].Field)
	assert.Equal(t, "high", q.Fields[0].Value)
	assert.Empty(t, q.FreeText)
}

func TestParseTaskQuery_MultiWordValue(t *testing.T) {
	q := app.ParseTaskQuery("status:in progress")
	require.Len(t, q.Fields, 1)
	assert.Equal(t, "status", q.Fields[0].Field)
	assert.Equal(t, "in progress", q.Fields[0].Value)
	assert.Empty(t, q.FreeText)
}

func TestParseTaskQuery_FieldGreedyValue(t *testing.T) {
	// Value is greedy: consumes all words up to the next field token or end.
	q := app.ParseTaskQuery("status:todo auth")
	require.Len(t, q.Fields, 1)
	assert.Equal(t, "status", q.Fields[0].Field)
	assert.Equal(t, "todo auth", q.Fields[0].Value)
	assert.Empty(t, q.FreeText)
}

func TestParseTaskQuery_FreeTextBeforeField(t *testing.T) {
	q := app.ParseTaskQuery("auth status:todo")
	require.Len(t, q.Fields, 1)
	assert.Equal(t, "status", q.Fields[0].Field)
	assert.Equal(t, "todo", q.Fields[0].Value)
	assert.Equal(t, "auth", q.FreeText)
}

func TestParseTaskQuery_MultipleFields(t *testing.T) {
	q := app.ParseTaskQuery("status:todo priority:high")
	require.Len(t, q.Fields, 2)
	assert.Equal(t, "status", q.Fields[0].Field)
	assert.Equal(t, "todo", q.Fields[0].Value)
	assert.Equal(t, "priority", q.Fields[1].Field)
	assert.Equal(t, "high", q.Fields[1].Value)
}

func TestParseTaskQuery_EmptyString(t *testing.T) {
	q := app.ParseTaskQuery("")
	assert.True(t, q.Empty())
	assert.Empty(t, q.Fields)
	assert.Empty(t, q.FreeText)
}

func TestParseTaskQuery_WhitespaceOnly(t *testing.T) {
	q := app.ParseTaskQuery("   ")
	assert.True(t, q.Empty())
}

func TestParseTaskQuery_FreeTextOnly(t *testing.T) {
	q := app.ParseTaskQuery("auth login")
	assert.Empty(t, q.Fields)
	assert.Equal(t, "auth login", q.FreeText)
}

func TestParseTaskQuery_UnknownField(t *testing.T) {
	// "foo:bar" is not a known field, so it becomes free text.
	q := app.ParseTaskQuery("foo:bar")
	assert.Empty(t, q.Fields)
	assert.Equal(t, "foo:bar", q.FreeText)
}

func TestParseTaskQuery_PartialFieldNoValue(t *testing.T) {
	// "status:" with no value should not produce a field filter.
	q := app.ParseTaskQuery("status:")
	assert.Empty(t, q.Fields)
	assert.True(t, q.Empty())
}

func TestParseTaskQuery_CaseInsensitiveField(t *testing.T) {
	q := app.ParseTaskQuery("Status:TODO")
	require.Len(t, q.Fields, 1)
	assert.Equal(t, "status", q.Fields[0].Field)
	assert.Equal(t, "todo", q.Fields[0].Value)
}

func TestParseTaskQuery_MultiWordValueFollowedByField(t *testing.T) {
	q := app.ParseTaskQuery("status:in progress priority:high")
	require.Len(t, q.Fields, 2)
	assert.Equal(t, "in progress", q.Fields[0].Value)
	assert.Equal(t, "high", q.Fields[1].Value)
}

// ── FilterTasks ─────────────────────────────────────────────────────────────

var testTasks = []app.TaskSummary{
	{ID: "1", Name: "Fix authentication bug", Status: "todo", Priority: "high"},
	{ID: "2", Name: "Update documentation", Status: "in progress", Priority: "normal"},
	{ID: "3", Name: "Refactor auth module", Status: "todo", Priority: "normal"},
	{ID: "4", Name: "Add login page", Status: "done", Priority: "low"},
	{ID: "5", Name: "Write unit tests", Status: "todo", Priority: "high"},
}

func TestFilterTasks_EmptyQuery(t *testing.T) {
	result := app.FilterTasks(testTasks, app.TaskQuery{})
	assert.Nil(t, result, "empty query should return nil (show all)")
}

func TestFilterTasks_FieldMatchOnly(t *testing.T) {
	q := app.ParseTaskQuery("status:todo")
	result := app.FilterTasks(testTasks, q)
	require.Len(t, result, 3)
	for _, r := range result {
		assert.Equal(t, "todo", r.Status)
	}
}

func TestFilterTasks_FieldMatchCaseInsensitive(t *testing.T) {
	q := app.ParseTaskQuery("status:TODO")
	result := app.FilterTasks(testTasks, q)
	require.Len(t, result, 3)
}

func TestFilterTasks_FieldSubstringMatch(t *testing.T) {
	// "in progress" contains "progress"
	q := app.ParseTaskQuery("status:progress")
	result := app.FilterTasks(testTasks, q)
	require.Len(t, result, 1)
	assert.Equal(t, "2", result[0].ID)
}

func TestFilterTasks_FuzzyMatchOnly(t *testing.T) {
	q := app.ParseTaskQuery("auth")
	result := app.FilterTasks(testTasks, q)
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
	result := app.FilterTasks(testTasks, q)
	require.NotEmpty(t, result)
	// "Refactor auth module" should score higher than "Fix authentication bug"
	// because "auth" is an exact substring in the name.
	assert.Equal(t, "3", result[0].ID, "exact substring match should rank first")
}

func TestFilterTasks_CombinedFieldAndFuzzy(t *testing.T) {
	// Free text before a field token: "auth status:todo"
	q := app.ParseTaskQuery("auth status:todo")
	result := app.FilterTasks(testTasks, q)
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
	result := app.FilterTasks(testTasks, q)
	assert.Nil(t, result)
}

func TestFilterTasks_NoMatchesFieldFilter(t *testing.T) {
	q := app.ParseTaskQuery("status:cancelled")
	result := app.FilterTasks(testTasks, q)
	assert.Nil(t, result)
}

func TestFilterTasks_MultipleFieldFilters(t *testing.T) {
	q := app.ParseTaskQuery("status:todo priority:high")
	result := app.FilterTasks(testTasks, q)
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
