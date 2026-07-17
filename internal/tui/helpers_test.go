// Package tui — shared test helpers.
package tui

import (
	"log/slog"
	"strings"
	"testing"
)

// testLogger returns a *slog.Logger that discards all output, for
// constructing a real *App in tests without a Discard target of its own.
func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.DiscardHandler)
}

// stripTviewTags removes tview colour/style tags (e.g. "[red]", "[#rrggbb]", "[-]")
// from a string so we can assert on visible text only.
func stripTviewTags(s string) string {
	var out strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '[':
			inTag = true
		case r == ']' && inTag:
			inTag = false
		case !inTag:
			out.WriteRune(r)
		}
	}
	return out.String()
}
