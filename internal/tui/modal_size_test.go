package tui

import (
	"testing"

	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func TestClamp(t *testing.T) {
	tests := []struct {
		name          string
		value, lo, hi int
		want          int
	}{
		{name: "below_low", value: 1, lo: 5, hi: 10, want: 5},
		{name: "within_range", value: 7, lo: 5, hi: 10, want: 7},
		{name: "above_high", value: 12, lo: 5, hi: 10, want: 10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, clamp(tc.value, tc.lo, tc.hi))
		})
	}
}

func TestModalScreenSize_Defaults(t *testing.T) {
	w, h := modalScreenSize(nil)
	assert.Equal(t, 80, w)
	assert.Equal(t, 24, h)

	w, h = modalScreenSize(&App{})
	assert.Equal(t, 80, w)
	assert.Equal(t, 24, h)
}

func TestModalScreenSize_FromPagesRect(t *testing.T) {
	pages := tview.NewPages()
	pages.SetRect(0, 0, 120, 40)

	w, h := modalScreenSize(&App{pages: pages})
	assert.Equal(t, 120, w)
	assert.Equal(t, 40, h)
}

func TestModalScreenSize_NonPositiveFallsBack(t *testing.T) {
	pages := tview.NewPages()
	pages.SetRect(0, 0, 0, -1)

	w, h := modalScreenSize(&App{pages: pages})
	assert.Equal(t, 80, w)
	assert.Equal(t, 24, h)
}
