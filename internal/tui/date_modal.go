// Package tui — date input modal.
package tui

import (
	"errors"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const pageDateModal = "date_modal"

// DateModalConfig configures a date input modal.
type DateModalConfig struct {
	// Title is displayed in the modal border.
	Title string
	// Initial pre-fills the input with an existing date in YYYY-MM-DD format.
	// An empty string leaves the field blank.
	Initial string
	// AllowClear adds a "Clear date" shortcut (Ctrl+D) that submits an empty
	// string, signalling the caller to remove the date.
	AllowClear bool
	// OnSubmit is called with the selected date in YYYY-MM-DD format, or ""
	// when the user clears the date (requires AllowClear: true).
	OnSubmit func(date string)
	// OnCancel is called when the user presses Esc.
	OnCancel func()
}

// validateDate checks that s is a valid YYYY-MM-DD date.
// Returns nil on success, an error describing the problem otherwise.
func validateDate(s string) error {
	if s == "" {
		return errors.New("date must not be empty")
	}
	if _, err := time.Parse(time.DateOnly, s); err != nil {
		return errors.New("use YYYY-MM-DD format (e.g. 2024-06-01)")
	}
	return nil
}

// ShowDateModal presents a centred date-input dialog over the current layout.
// Enter submits with YYYY-MM-DD validation.  Esc cancels.  When AllowClear is
// true, Ctrl+D submits an empty string to clear the date.  It calls
// SetModalActive(true) on show and SetModalActive(false) when dismissed.
func ShowDateModal(a *App, cfg DateModalConfig) {
	dismiss := func() {
		a.SetModalActive(false)
		a.pages.RemovePage(pageDateModal)
		a.restoreDefaultHelp()
	}

	errLabel := tview.NewTextView().
		SetDynamicColors(true).
		SetText("")
	errLabel.SetBorder(false)

	input := tview.NewInputField().
		SetLabel(" ").
		SetText(cfg.Initial).
		SetPlaceholder("YYYY-MM-DD").
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetFieldTextColor(tcell.ColorWhite).
		SetLabelColor(ColorTextMuted)

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			val := input.GetText()
			if err := validateDate(val); err != nil {
				errLabel.SetText(tagColor(ColorStatusError) + " " + tview.Escape(err.Error()) + "[-]")
				return nil
			}
			dismiss()
			if cfg.OnSubmit != nil {
				cfg.OnSubmit(val)
			}
			return nil
		case tcell.KeyEscape:
			dismiss()
			if cfg.OnCancel != nil {
				cfg.OnCancel()
			}
			return nil
		case tcell.KeyCtrlD:
			if cfg.AllowClear {
				dismiss()
				if cfg.OnSubmit != nil {
					cfg.OnSubmit("")
				}
				return nil
			}
		}
		return event
	})

	frame := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(input, 1, 0, true).
		AddItem(errLabel, 1, 0, false)
	frame.SetBorder(true).
		SetTitle(" " + cfg.Title + " ").
		SetBorderColor(ColorBorderFocused).
		SetTitleColor(ColorTitleFocused)

	modal := centreModal(frame, 44, 5)

	if cfg.AllowClear {
		a.footer.SetHelp("Enter:confirm", "Ctrl+D:clear", "Esc:cancel")
	} else {
		a.footer.SetHelp("Enter:confirm", "Esc:cancel")
	}
	a.SetModalActive(true)
	a.pages.AddPage(pageDateModal, modal, true, true)
	a.tviewApp.SetFocus(input)
}
