// Package app -- pure ranking for the assignee picker.
//
// SortMembersByRecency surfaces likely candidates first: members the user has
// assigned recently float to the top, in most-recent-first order, ahead of
// everyone else (who keep their original relative order).
package app

// SortMembersByRecency reorders members so that any member whose ID appears
// in recentIDs comes first, ordered to match recentIDs (most-recent first).
// Remaining members keep their original relative order. members and recentIDs
// are not mutated; a new slice is returned.
func SortMembersByRecency(members []MemberSummary, recentIDs []int) []MemberSummary {
	if len(members) == 0 || len(recentIDs) == 0 {
		return members
	}

	byID := make(map[int]MemberSummary, len(members))
	for _, m := range members {
		byID[m.ID] = m
	}

	result := make([]MemberSummary, 0, len(members))
	seen := make(map[int]struct{}, len(recentIDs))
	for _, id := range recentIDs {
		if m, ok := byID[id]; ok {
			result = append(result, m)
			seen[id] = struct{}{}
		}
	}
	for _, m := range members {
		if _, ok := seen[m.ID]; !ok {
			result = append(result, m)
		}
	}
	return result
}
