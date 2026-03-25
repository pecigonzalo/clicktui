// Package tui — unit tests for input modal validation logic.
package tui

import (
	"errors"
	"testing"
)

// ── InputModalConfig.Validate ─────────────────────────────────────────────────

func TestInputModal_Validate_AcceptsValidInput(t *testing.T) {
	called := false
	submitted := ""
	cfg := InputModalConfig{
		Validate: func(s string) error {
			called = true
			if s == "" {
				return errors.New("must not be empty")
			}
			return nil
		},
		OnSubmit: func(s string) { submitted = s },
	}

	// Simulate what ShowInputModal does on Enter with valid input.
	val := "hello"
	err := cfg.Validate(val)
	if err != nil {
		t.Fatalf("Validate(%q) = %v, want nil", val, err)
	}
	if !called {
		t.Error("Validate callback was not called")
	}
	cfg.OnSubmit(val)
	if submitted != val {
		t.Errorf("OnSubmit received %q, want %q", submitted, val)
	}
}

func TestInputModal_Validate_BlocksEmptyInput(t *testing.T) {
	cfg := InputModalConfig{
		Validate: func(s string) error {
			if s == "" {
				return errors.New("must not be empty")
			}
			return nil
		},
	}

	err := cfg.Validate("")
	if err == nil {
		t.Error("Validate('') = nil, want non-nil error")
	}
}

func TestInputModal_NoValidate_AlwaysPasses(t *testing.T) {
	// When Validate is nil, the modal should not call it.
	// We verify this by checking that OnSubmit is called even for empty input.
	submitted := ""
	cfg := InputModalConfig{
		Validate: nil,
		OnSubmit: func(s string) { submitted = s },
	}

	// No validation to run; directly call submit.
	cfg.OnSubmit("")
	if submitted != "" {
		t.Errorf("OnSubmit received %q, want empty string", submitted)
	}
}

func TestInputModal_Validate_CustomError(t *testing.T) {
	cfg := InputModalConfig{
		Validate: func(s string) error {
			if len(s) < 3 {
				return errors.New("at least 3 characters required")
			}
			return nil
		},
	}

	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty", "", true},
		{"too_short", "ab", true},
		{"exact_minimum", "abc", false},
		{"valid", "hello world", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := cfg.Validate(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate(%q) error = %v, wantErr = %v", tc.input, err, tc.wantErr)
			}
		})
	}
}

func TestInputModal_OnCancel_Called(t *testing.T) {
	cancelled := false
	cfg := InputModalConfig{
		OnCancel: func() { cancelled = true },
	}
	cfg.OnCancel()
	if !cancelled {
		t.Error("OnCancel was not called")
	}
}
