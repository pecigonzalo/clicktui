// Package tui — single-line text input modal.
package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const pageInputModal = "input_modal"

// InputModalConfig configures a single-line text input modal.
type InputModalConfig struct {
	// Title is displayed in the modal border.
	Title string
	// Placeholder is the greyed-out hint text shown when the field is empty.
	Placeholder string
	// Initial pre-fills the input field with an existing value.
	Initial string
	// Validate is called on submit; return a non-nil error to block submission
	// and show an inline error message. May be nil to skip validation.
	Validate func(string) error
	// OnSubmit is called with the entered text when the user presses Enter and
	// validation passes.
	OnSubmit func(string)
	// OnCancel is called when the user presses Esc.
	OnCancel func()
}

// ShowInputModal presents a centred single-line input dialog over the current
// layout.  It calls SetModalActive(true) on show and SetModalActive(false) when
// the modal is dismissed (either via submit or cancel).
func ShowInputModal(a *App, cfg InputModalConfig) {
	dismiss := func() {
		a.SetModalActive(false)
		a.pages.RemovePage(pageInputModal)
		a.restoreDefaultHelp()
	}

	// Error label displayed below the input when validation fails.
	errLabel := tview.NewTextView().
		SetDynamicColors(true).
		SetText("")
	errLabel.SetBorder(false)

	input := tview.NewInputField().
		SetLabel(" ").
		SetText(cfg.Initial).
		SetPlaceholder(cfg.Placeholder).
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetFieldTextColor(tcell.ColorWhite).
		SetLabelColor(ColorTextMuted)

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			val := input.GetText()
			if cfg.Validate != nil {
				if err := cfg.Validate(val); err != nil {
					errLabel.SetText(tagColor(ColorStatusError) + " " + tview.Escape(err.Error()) + "[-]")
					return nil
				}
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

	modal := centreModal(frame, 52, 5)

	a.footer.SetHelp("Enter:confirm", "Esc:cancel")
	a.SetModalActive(true)
	a.pages.AddPage(pageInputModal, modal, true, true)
	a.tviewApp.SetFocus(input)
}
