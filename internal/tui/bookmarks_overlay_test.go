// Package tui — unit tests for bookmarks overlay logic.
package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── pageBookmarksOverlay constant ─────────────────────────────────────────────

func TestPageBookmarksOverlay_Constant(t *testing.T) {
	t.Parallel()
	require.NotEmpty(t, pageBookmarksOverlay)
	// Verify it does not collide with other page names.
	assert.NotEqual(t, pageBookmarksOverlay, pageStatusPicker)
	assert.NotEqual(t, pageBookmarksOverlay, pageSelectModal)
	assert.NotEqual(t, pageBookmarksOverlay, pageMain)
}

// ── test helpers ──────────────────────────────────────────────────────────────

// newTestAppWithFooter creates a minimal *App with a Footer wired so we can
// observe status messages without a live tview.Application.
func newTestAppWithFooter(t *testing.T) *App {
	t.Helper()
	return &App{
		footer: newFooter(),
	}
}
