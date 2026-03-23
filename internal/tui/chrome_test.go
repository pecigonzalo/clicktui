// Package tui — unit tests for chrome helper functions.
package tui

import (
	"strings"
	"testing"
)

// ── state placeholder helpers ─────────────────────────────────────────────────

func TestErrorText(t *testing.T) {
	got := errorText("something went wrong")
	if !strings.Contains(got, "something went wrong") {
		t.Errorf("errorText = %q, expected to contain the message", got)
	}
	if !strings.Contains(got, "[red]") {
		t.Errorf("errorText = %q, expected red colour tag", got)
	}
	if !strings.Contains(got, "✗") {
		t.Errorf("errorText = %q, expected ✗ symbol", got)
	}
}

func TestEmptyText(t *testing.T) {
	got := emptyText("No tasks found")
	if !strings.Contains(got, "No tasks found") {
		t.Errorf("emptyText = %q, expected to contain the message", got)
	}
	// Should include a colour tag.
	if !strings.Contains(got, "[") {
		t.Errorf("emptyText = %q, expected at least one colour tag", got)
	}
}

// ── Footer.SetHelp ────────────────────────────────────────────────────────────

func TestFooterSetHelp_FormatsKeyAction(t *testing.T) {
	f := newFooter()
	f.SetHelp("Tab:next pane", "q:quit")
	text := f.GetText(false)
	if !strings.Contains(text, "Tab") {
		t.Errorf("footer text %q should contain 'Tab'", text)
	}
	if !strings.Contains(text, "next pane") {
		t.Errorf("footer text %q should contain 'next pane'", text)
	}
	if !strings.Contains(text, "quit") {
		t.Errorf("footer text %q should contain 'quit'", text)
	}
}

func TestFooterSetHelp_NoPairs(t *testing.T) {
	f := newFooter()
	f.SetHelp() // empty — should not panic
	_ = f.GetText(false)
}

func TestFooterSetHelp_PairWithoutColon(t *testing.T) {
	f := newFooter()
	// Pair without colon should render the raw string without panic.
	f.SetHelp("no-colon-here")
	text := f.GetText(false)
	if !strings.Contains(text, "no-colon-here") {
		t.Errorf("footer text %q should contain 'no-colon-here'", text)
	}
}

// ── Footer status setters ─────────────────────────────────────────────────────

func TestFooterSetStatusLoading(t *testing.T) {
	f := newFooter()
	f.SetStatusLoading("Loading %s", "things")
	text := f.GetText(false)
	if !strings.Contains(text, "Loading things") {
		t.Errorf("footer text %q should contain 'Loading things'", text)
	}
}

func TestFooterSetStatusError(t *testing.T) {
	f := newFooter()
	f.SetStatusError("oops: %v", "disk full")
	text := f.GetText(false)
	if !strings.Contains(text, "oops: disk full") {
		t.Errorf("footer text %q should contain 'oops: disk full'", text)
	}
}

func TestFooterSetStatusReady(t *testing.T) {
	f := newFooter()
	f.SetStatusReady("Ready")
	text := f.GetText(false)
	if !strings.Contains(text, "Ready") {
		t.Errorf("footer text %q should contain 'Ready'", text)
	}
}
