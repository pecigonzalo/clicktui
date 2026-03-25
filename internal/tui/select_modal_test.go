// Package tui — unit tests for select modal option helpers.
package tui

import (
	"testing"
)

// ── SelectedValues — pure helper ─────────────────────────────────────────────

func TestSelectedValues_NoneSelected(t *testing.T) {
	opts := []SelectOption{
		{Label: "A", Value: "a"},
		{Label: "B", Value: "b"},
		{Label: "C", Value: "c"},
	}
	got := SelectedValues(opts)
	if len(got) != 0 {
		t.Errorf("SelectedValues() = %v, want empty slice", got)
	}
}

func TestSelectedValues_AllSelected(t *testing.T) {
	opts := []SelectOption{
		{Label: "A", Value: "a", Selected: true},
		{Label: "B", Value: "b", Selected: true},
	}
	got := SelectedValues(opts)
	if len(got) != 2 {
		t.Fatalf("SelectedValues() = %v (len %d), want 2 items", got, len(got))
	}
	if got[0] != "a" || got[1] != "b" {
		t.Errorf("SelectedValues() = %v, want [a b]", got)
	}
}

func TestSelectedValues_SomeSelected(t *testing.T) {
	opts := []SelectOption{
		{Label: "A", Value: "a"},
		{Label: "B", Value: "b", Selected: true},
		{Label: "C", Value: "c"},
		{Label: "D", Value: "d", Selected: true},
	}
	got := SelectedValues(opts)
	if len(got) != 2 {
		t.Fatalf("SelectedValues() = %v (len %d), want 2 items", got, len(got))
	}
	if got[0] != "b" || got[1] != "d" {
		t.Errorf("SelectedValues() = %v, want [b d]", got)
	}
}

func TestSelectedValues_EmptyOptions(t *testing.T) {
	got := SelectedValues(nil)
	if len(got) != 0 {
		t.Errorf("SelectedValues(nil) = %v, want empty", got)
	}
}

// ── SelectOption — field correctness ────────────────────────────────────────

func TestSelectOption_DefaultSelectedFalse(t *testing.T) {
	opt := SelectOption{Label: "Foo", Value: "foo"}
	if opt.Selected {
		t.Error("SelectOption.Selected should default to false")
	}
}

func TestSelectOption_AllFields(t *testing.T) {
	opt := SelectOption{Label: "My Label", Value: "my-value", Selected: true}
	if opt.Label != "My Label" {
		t.Errorf("Label = %q, want 'My Label'", opt.Label)
	}
	if opt.Value != "my-value" {
		t.Errorf("Value = %q, want 'my-value'", opt.Value)
	}
	if !opt.Selected {
		t.Error("Selected should be true")
	}
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
	if len(received) != 1 || received[0] != "alpha" {
		t.Errorf("OnSubmit received %v, want [alpha]", received)
	}
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
	if len(received) != 2 {
		t.Fatalf("OnSubmit received %v (len %d), want 2 items", received, len(received))
	}
	if received[0] != "a" || received[1] != "c" {
		t.Errorf("OnSubmit received %v, want [a c]", received)
	}
}

func TestSelectModal_OnCancel_Called(t *testing.T) {
	cancelled := false
	cfg := SelectModalConfig{
		OnCancel: func() { cancelled = true },
	}
	cfg.OnCancel()
	if !cancelled {
		t.Error("OnCancel was not called")
	}
}
