// Package tui — external editor integration helpers.
package tui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rivo/tview"
)

// errUnchanged indicates editor output is identical to the original content.
var errUnchanged = errors.New("unchanged")

// resolveEditor returns the editor command from $VISUAL or $EDITOR (in that
// order), split on whitespace. Returns nil if neither is set.
func resolveEditor() []string {
	if visual := strings.TrimSpace(os.Getenv("VISUAL")); visual != "" {
		return strings.Fields(visual)
	}
	if editor := strings.TrimSpace(os.Getenv("EDITOR")); editor != "" {
		return strings.Fields(editor)
	}
	return nil
}

// openInEditor suspends the tview app, opens current content in the configured
// editor, then returns updated content.
func openInEditor(tviewApp *tview.Application, current string) (string, error) {
	if tviewApp == nil {
		return "", errors.New("tview app is nil")
	}

	argv := resolveEditor()
	if len(argv) == 0 {
		return "", errors.New("editor not configured")
	}

	var (
		tmpPath string
		runErr  error
	)

	suspended := tviewApp.Suspend(func() {
		tmp, err := os.CreateTemp("", "clicktui-*.md")
		if err != nil {
			runErr = fmt.Errorf("create temp file: %w", err)
			return
		}
		tmpPath = tmp.Name()

		if _, err := tmp.WriteString(current); err != nil {
			runErr = fmt.Errorf("write temp file: %w", err)
			_ = tmp.Close()
			return
		}
		if err := tmp.Close(); err != nil {
			runErr = fmt.Errorf("close temp file: %w", err)
			return
		}

		cmd := exec.Command(argv[0], append(argv[1:], tmpPath)...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		runErr = cmd.Run()
	})
	if !suspended {
		return "", errors.New("unable to suspend terminal UI")
	}
	if tmpPath != "" {
		defer func() {
			_ = os.Remove(tmpPath)
		}()
	}
	if runErr != nil {
		return "", runErr
	}

	updatedBytes, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", fmt.Errorf("read temp file: %w", err)
	}
	updated := string(updatedBytes)

	if normalizeForComparison(updated) == normalizeForComparison(current) {
		return "", errUnchanged
	}
	return updated, nil
}

func normalizeForComparison(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.TrimSuffix(s, "\n")
}
