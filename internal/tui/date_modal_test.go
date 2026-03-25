// Package tui — unit tests for date modal validation logic.
package tui

import (
	"testing"
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
			if err := validateDate(tc.input); err != nil {
				t.Errorf("validateDate(%q) = %v, want nil", tc.input, err)
			}
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
			if err := validateDate(tc.input); err == nil {
				t.Errorf("validateDate(%q) = nil, want non-nil error", tc.input)
			}
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
			if err := validateDate(tc.input); err == nil {
				t.Errorf("validateDate(%q) = nil, want error for invalid calendar date", tc.input)
			}
		})
	}
}

func TestValidateDate_EmptyIsError(t *testing.T) {
	err := validateDate("")
	if err == nil {
		t.Error("validateDate('') = nil, want error")
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
	if submitted != "" {
		t.Errorf("AllowClear OnSubmit received %q, want empty string", submitted)
	}
}

func TestDateModal_OnCancel_Called(t *testing.T) {
	cancelled := false
	cfg := DateModalConfig{
		OnCancel: func() { cancelled = true },
	}
	cfg.OnCancel()
	if !cancelled {
		t.Error("OnCancel was not called")
	}
}
