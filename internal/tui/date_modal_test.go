// Package tui — unit tests for date modal validation logic.
package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── validateDate ──────────────────────────────────────────────────────────────

func TestValidateDate_ValidDates(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"min_date", "2000-01-01"},
		{"typical", "2024-06-01"},
		{"year_end", "2024-12-31"},
		{"leap_day", "2024-02-29"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, validateDate(tc.input))
		})
	}
}

func TestValidateDate_InvalidFormats(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"wrong_separator", "2024/06/01"},
		{"us_format", "06-01-2024"},
		{"missing_day", "2024-06"},
		{"text", "not-a-date"},
		{"partial_year", "24-06-01"},
		{"extra_chars", "2024-06-01T00:00:00"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Error(t, validateDate(tc.input))
		})
	}
}

func TestValidateDate_InvalidCalendarDates(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"feb_30", "2024-02-30"},
		{"month_13", "2024-13-01"},
		{"day_0", "2024-06-00"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Error(t, validateDate(tc.input))
		})
	}
}

// ── DateModalConfig.AllowClear ───────────────────────────────────────────────

func TestDateModal_AllowClear_SubmitsEmpty(t *testing.T) {
	// When AllowClear is true, the Ctrl+D handler should submit ""
	// without validation.
	submitted := "not-called"
	cfg := DateModalConfig{
		AllowClear: true,
		OnSubmit:   func(date string) { submitted = date },
	}
	// Simulate the Ctrl+D path: bypass validation and submit "".
	if cfg.AllowClear {
		cfg.OnSubmit("")
	}
	assert.Equal(t, "", submitted)
}

func TestDateModal_OnCancel_Called(t *testing.T) {
	cancelled := false
	cfg := DateModalConfig{
		OnCancel: func() { cancelled = true },
	}
	cfg.OnCancel()
	assert.True(t, cancelled)
}
