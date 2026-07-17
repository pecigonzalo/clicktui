// Package tui — unit tests for the help overlay.
package tui

import (
	"strings"
	"testing"

	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageHelpOverlay_Constant(t *testing.T) {
	t.Parallel()
	require.NotEmpty(t, pageHelpOverlay)
	assert.NotEqual(t, pageHelpOverlay, pageBookmarksOverlay)
	assert.NotEqual(t, pageHelpOverlay, pageStatusPicker)
	assert.NotEqual(t, pageHelpOverlay, pageMain)
}

func TestRenderHelpText_ContainsEverySection(t *testing.T) {
	t.Parallel()
	raw := renderHelpText()
	text := stripTviewTags(raw)
	for _, section := range helpSections {
		assert.Contains(t, text, strings.ToUpper(section.title), "missing section header %q", section.title)
		for _, b := range section.binds {
			// Keys containing "[" are escaped for tview (as "[["), which the
			// naive stripTviewTags test helper can't unescape, so check the
			// raw string for those instead of the stripped one.
			if strings.Contains(b[0], "[") {
				assert.Contains(t, raw, tview.Escape(b[0]), "missing escaped key %q from section %q", b[0], section.title)
			} else {
				assert.Contains(t, text, b[0], "missing key %q from section %q", b[0], section.title)
			}
			assert.Contains(t, text, b[1], "missing description %q from section %q", b[1], section.title)
		}
	}
}

func TestRenderHelpText_MentionsHelpToggle(t *testing.T) {
	t.Parallel()
	text := stripTviewTags(renderHelpText())
	assert.Contains(t, text, "?")
	assert.Contains(t, text, "Toggle this help")
}

func TestPadRight(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "ab  ", padRight("ab", 4))
	assert.Equal(t, "abcd", padRight("abcd", 4))
	assert.Equal(t, "abcdef", padRight("abcdef", 4))
	assert.Equal(t, "", padRight("", 0))
}
