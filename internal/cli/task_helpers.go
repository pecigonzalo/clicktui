// Package cli — shared helpers for task mutation commands.
package cli

import (
	"fmt"
	"time"
)

// parsePriority converts a priority name string to the integer value expected
// by the ClickUp API (urgent=1, high=2, normal=3, low=4, none/""=0).
// Returns an error for unrecognised names.
func parsePriority(name string) (int, error) {
	switch name {
	case "urgent":
		return 1, nil
	case "high":
		return 2, nil
	case "normal":
		return 3, nil
	case "low":
		return 4, nil
	case "none", "":
		return 0, nil
	default:
		return 0, fmt.Errorf("unknown priority %q: must be urgent, high, normal, low, or none", name)
	}
}

// parseDueDate converts a YYYY-MM-DD date string to an epoch-milliseconds string
// suitable for the ClickUp API.  An empty input returns "" (no due date).
func parseDueDate(date string) (string, error) {
	if date == "" {
		return "", nil
	}
	t, err := time.ParseInLocation(time.DateOnly, date, time.UTC)
	if err != nil {
		return "", fmt.Errorf("invalid due date %q: expected YYYY-MM-DD", date)
	}
	ms := t.UnixMilli()
	return fmt.Sprintf("%d", ms), nil
}
