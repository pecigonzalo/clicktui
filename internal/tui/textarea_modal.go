// Package tui — multi-line text area modal.
package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const pageTextAreaModal = "textarea_modal"

// TextAreaModalConfig configures a multi-line text area modal.
type TextAreaModalConfig struct {
	// Title is displayed in the modal border.
	Title string
	// Initial pre-fills the text area with existing content.
	Initial string
	// OnSubmit is called with the entered text when the user saves (Ctrl+S).
	OnSubmit func(string)
	// OnCancel is called when the user presses Esc.
	OnCancel func()
}

// ShowTextAreaModal presents a centred multi-line text editor dialog over the
// current layout.  Ctrl+S saves (submits), Esc cancels.  It calls
// SetModalActive(true) on show and SetModalActive(false) when dismissed.
func ShowTextAreaModal(a *App, cfg TextAreaModalConfig) {
	dismiss := func() {
		a.SetModalActive(false)
		a.pages.RemovePage(pageTextAreaModal)
		a.restoreDefaultHelp()
	}

	area := tview.NewTextArea().
		SetText(cfg.Initial, false).
		SetPlaceholder("Enter text…")
	area.SetBorder(false)
	area.SetTextStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite))

	area.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			dismiss()
			if cfg.OnCancel != nil {
				cfg.OnCancel()
			}
			return nil
		case tcell.KeyCtrlS:
			val := area.GetText()
			dismiss()
			if cfg.OnSubmit != nil {
				cfg.OnSubmit(val)
			}
			return nil
		}
		return event
	})

	frame := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(area, 0, 1, true)
	frame.SetBorder(true).
		SetTitle(" " + cfg.Title + " ").
		SetBorderColor(ColorBorderFocused).
		SetTitleColor(ColorTitleFocused)

	screenW, screenH := modalScreenSize(a)
	modalW := clamp(screenW*70/100, 60, 110)
	modalH := clamp(screenH*50/100, 12, 30)
	modal := centreModal(frame, modalW, modalH)

	a.footer.SetHelp("Ctrl+S:save", "Esc:cancel")
	a.SetModalActive(true)
	a.pages.AddPage(pageTextAreaModal, modal, true, true)
	a.tviewApp.SetFocus(area)
}
