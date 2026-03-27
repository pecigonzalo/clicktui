// Package tui — unit tests for chrome helper functions.
package tui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── state placeholder helpers ─────────────────────────────────────────────────

func TestErrorText(t *testing.T) {
	got := errorText("something went wrong")
	assert.True(t, strings.Contains(got, "something went wrong"))
	assert.True(t, strings.Contains(got, "[red]"))
	assert.True(t, strings.Contains(got, "✗"))
}

func TestEmptyText(t *testing.T) {
	got := emptyText("No tasks found")
	assert.True(t, strings.Contains(got, "No tasks found"))
	assert.True(t, strings.Contains(got, "["))
}

// ── Footer.SetHelp ────────────────────────────────────────────────────────────

func TestFooterSetHelp_FormatsKeyAction(t *testing.T) {
	f := newTestAppWithFooter(t).footer
	f.SetHelp("Tab:next pane", "q:quit")
	text := f.GetText(false)
	assert.True(t, strings.Contains(text, "Tab"))
	assert.True(t, strings.Contains(text, "next pane"))
	assert.True(t, strings.Contains(text, "quit"))
}

func TestFooterSetHelp_NoPairs(t *testing.T) {
	f := newTestAppWithFooter(t).footer
	f.SetHelp() // empty — should not panic
	_ = f.GetText(false)
}

func TestFooterSetHelp_PairWithoutColon(t *testing.T) {
	f := newTestAppWithFooter(t).footer
	// Pair without colon should render the raw string without panic.
	f.SetHelp("no-colon-here")
	text := f.GetText(false)
	assert.True(t, strings.Contains(text, "no-colon-here"))
}

// ── Footer status setters ─────────────────────────────────────────────────────

func TestFooterSetStatusLoading(t *testing.T) {
	f := newTestAppWithFooter(t).footer
	f.SetStatusLoading("Loading %s", "things")
	text := f.GetText(false)
	assert.True(t, strings.Contains(text, "Loading things"))
}

func TestFooterSetStatusError(t *testing.T) {
	f := newTestAppWithFooter(t).footer
	f.SetStatusError("oops: %v", "disk full")
	text := f.GetText(false)
	assert.True(t, strings.Contains(text, "oops: disk full"))
}

func TestFooterSetStatusReady(t *testing.T) {
	f := newTestAppWithFooter(t).footer
	f.SetStatusReady("Ready")
	text := f.GetText(false)
	require.True(t, strings.Contains(text, "Ready"))
}
