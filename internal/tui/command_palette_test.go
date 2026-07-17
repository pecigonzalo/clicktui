// Package tui — unit tests for the command palette.
package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pecigonzalo/clicktui/internal/app"
)

func TestPageCommandPalette_Constant(t *testing.T) {
	t.Parallel()
	require.NotEmpty(t, pageCommandPalette)
	assert.NotEqual(t, pageCommandPalette, pageHelpOverlay)
	assert.NotEqual(t, pageCommandPalette, pageBookmarksOverlay)
	assert.NotEqual(t, pageCommandPalette, pageMain)
}

func TestBreadcrumbFor_FolderlessList(t *testing.T) {
	t.Parallel()
	got := breadcrumbFor(app.ListRef{SpaceName: "Marketing"})
	assert.Equal(t, "Marketing", got)
}

func TestBreadcrumbFor_ListInFolder(t *testing.T) {
	t.Parallel()
	got := breadcrumbFor(app.ListRef{SpaceName: "Engineering", FolderName: "Sprints"})
	assert.Contains(t, got, "Engineering")
	assert.Contains(t, got, "Sprints")
}
