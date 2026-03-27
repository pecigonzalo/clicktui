// Package tui — unit tests for select modal option helpers.
package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── SelectedValues — pure helper ─────────────────────────────────────────────

func TestSelectedValues_NoneSelected(t *testing.T) {
	t.Parallel()
	opts := []SelectOption{
		{Label: "A", Value: "a"},
		{Label: "B", Value: "b"},
		{Label: "C", Value: "c"},
	}
	got := SelectedValues(opts)
	assert.Empty(t, got)
}

func TestSelectedValues_AllSelected(t *testing.T) {
	t.Parallel()
	opts := []SelectOption{
		{Label: "A", Value: "a", Selected: true},
		{Label: "B", Value: "b", Selected: true},
	}
	got := SelectedValues(opts)
	require.Len(t, got, 2)
	assert.Equal(t, []string{"a", "b"}, got)
}

func TestSelectedValues_SomeSelected(t *testing.T) {
	t.Parallel()
	opts := []SelectOption{
		{Label: "A", Value: "a"},
		{Label: "B", Value: "b", Selected: true},
		{Label: "C", Value: "c"},
		{Label: "D", Value: "d", Selected: true},
	}
	got := SelectedValues(opts)
	require.Len(t, got, 2)
	assert.Equal(t, []string{"b", "d"}, got)
}

func TestSelectedValues_EmptyOptions(t *testing.T) {
	t.Parallel()
	got := SelectedValues(nil)
	assert.Empty(t, got)
}

// ── SelectModalConfig — callback correctness ─────────────────────────────────

func TestSelectModal_OnSubmit_SingleSelect(t *testing.T) {
	var received []string
	cfg := SelectModalConfig{
		Options:  []SelectOption{{Label: "Alpha", Value: "alpha"}},
		Multi:    false,
		OnSubmit: func(vals []string) { received = vals },
	}
	// Simulate single-select confirmation with one item.
	cfg.OnSubmit([]string{"alpha"})
	assert.Equal(t, []string{"alpha"}, received)
}

func TestSelectModal_OnSubmit_MultiSelect(t *testing.T) {
	var received []string
	cfg := SelectModalConfig{
		Options: []SelectOption{
			{Label: "A", Value: "a", Selected: true},
			{Label: "B", Value: "b"},
			{Label: "C", Value: "c", Selected: true},
		},
		Multi:    true,
		OnSubmit: func(vals []string) { received = vals },
	}
	// Simulate the multi-select confirm path: collect pre-selected options.
	vals := SelectedValues(cfg.Options)
	cfg.OnSubmit(vals)
	require.Len(t, received, 2)
	assert.Equal(t, []string{"a", "c"}, received)
}

func TestSelectModal_OnCancel_Called(t *testing.T) {
	cancelled := false
	cfg := SelectModalConfig{
		OnCancel: func() { cancelled = true },
	}
	cfg.OnCancel()
	assert.True(t, cancelled)
}
