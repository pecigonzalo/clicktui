package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pecigonzalo/clicktui/internal/app"
)

func TestSortMembersByRecency_EmptyRecent_ReturnsUnchanged(t *testing.T) {
	members := []app.MemberSummary{{ID: 1, Username: "alice"}, {ID: 2, Username: "bob"}}
	got := app.SortMembersByRecency(members, nil)
	assert.Equal(t, members, got)
}

func TestSortMembersByRecency_EmptyMembers_ReturnsNil(t *testing.T) {
	got := app.SortMembersByRecency(nil, []int{1, 2})
	assert.Nil(t, got)
}

func TestSortMembersByRecency_PutsRecentFirstInOrder(t *testing.T) {
	members := []app.MemberSummary{
		{ID: 1, Username: "alice"},
		{ID: 2, Username: "bob"},
		{ID: 3, Username: "carol"},
		{ID: 4, Username: "dave"},
	}
	// Most-recent-first: 3, then 1. Neither 2 nor 4 were ever assigned.
	got := app.SortMembersByRecency(members, []int{3, 1})

	assert.Len(t, got, 4)
	assert.Equal(t, 3, got[0].ID)
	assert.Equal(t, 1, got[1].ID)
	// Remaining members keep their original relative order.
	assert.Equal(t, 2, got[2].ID)
	assert.Equal(t, 4, got[3].ID)
}

func TestSortMembersByRecency_IgnoresRecentIDsNotInMembers(t *testing.T) {
	members := []app.MemberSummary{{ID: 1, Username: "alice"}, {ID: 2, Username: "bob"}}
	got := app.SortMembersByRecency(members, []int{999, 2})
	assert.Equal(t, []app.MemberSummary{
		{ID: 2, Username: "bob"},
		{ID: 1, Username: "alice"},
	}, got)
}

func TestSortMembersByRecency_DoesNotMutateInputs(t *testing.T) {
	members := []app.MemberSummary{{ID: 1, Username: "alice"}, {ID: 2, Username: "bob"}}
	original := append([]app.MemberSummary(nil), members...)
	_ = app.SortMembersByRecency(members, []int{2})
	assert.Equal(t, original, members)
}
