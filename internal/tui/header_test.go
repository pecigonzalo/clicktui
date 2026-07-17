// Package tui — unit tests for the header bar.
package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeader_SetProfile_ShowsProfileName(t *testing.T) {
	h := newHeader()
	h.SetProfile("work")
	assert.Contains(t, stripTviewTags(h.GetText(false)), "profile: work")
}

func TestHeader_SetProfile_DoesNotPanicOnBrackets(t *testing.T) {
	h := newHeader()
	// A profile name containing '[' must not be mistaken for a tview colour
	// tag or panic — SetProfile escapes it via tview.Escape.
	assert.NotPanics(t, func() { h.SetProfile("we[ird]") })
}

func TestHeader_ShowsAppName(t *testing.T) {
	h := newHeader()
	h.SetProfile("default")
	assert.Contains(t, stripTviewTags(h.GetText(false)), "clicktui")
}
