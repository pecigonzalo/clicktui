package app_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pecigonzalo/clicktui/internal/app"
	"github.com/pecigonzalo/clicktui/internal/config"
)

// ── fakeConfigLoader ──────────────────────────────────────────────────────────

// fakeConfigLoader is an in-memory ConfigLoader for tests.
type fakeConfigLoader struct {
	cfg     *config.Config
	loadErr error
	saveErr error
}

func (f *fakeConfigLoader) Load() (*config.Config, error) {
	if f.loadErr != nil {
		return nil, f.loadErr
	}
	return f.cfg, nil
}

func (f *fakeConfigLoader) Save(cfg *config.Config) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.cfg = cfg
	return nil
}

// newFakeLoader returns a loader pre-populated with a default profile.
func newFakeLoader() *fakeConfigLoader {
	cfg := config.New()
	cfg.SetProfile(&config.Profile{
		Name:       config.DefaultProfile(),
		AuthMethod: config.AuthMethodPersonalToken,
	})
	return &fakeConfigLoader{cfg: cfg}
}

// ── GetSortPreference ─────────────────────────────────────────────────────────

func TestGetSortPreference_DefaultsWhenNoPreference(t *testing.T) {
	svc := app.NewUIStateServiceWithLoader(newFakeLoader())

	field, asc := svc.GetSortPreference("default")

	assert.Equal(t, "", field, "field should be empty when no preference saved")
	assert.False(t, asc, "ascending should be false when no preference saved")
}

func TestGetSortPreference_ProfileNotFound_ReturnsDefaults(t *testing.T) {
	svc := app.NewUIStateServiceWithLoader(newFakeLoader())

	field, asc := svc.GetSortPreference("nonexistent")

	assert.Equal(t, "", field)
	assert.False(t, asc)
}

func TestGetSortPreference_LoadError_ReturnsDefaults(t *testing.T) {
	loader := &fakeConfigLoader{loadErr: errors.New("disk error")}
	svc := app.NewUIStateServiceWithLoader(loader)

	field, asc := svc.GetSortPreference("default")

	assert.Equal(t, "", field)
	assert.False(t, asc)
}

// ── SetSortPreference ─────────────────────────────────────────────────────────

func TestSetSortPreference_PersistsFieldAndDirection(t *testing.T) {
	loader := newFakeLoader()
	svc := app.NewUIStateServiceWithLoader(loader)

	err := svc.SetSortPreference("default", "priority", true)
	require.NoError(t, err)

	field, asc := svc.GetSortPreference("default")
	assert.Equal(t, "priority", field)
	assert.True(t, asc)
}

func TestSetSortPreference_RoundTrip_AllFields(t *testing.T) {
	cases := []struct {
		field string
		asc   bool
	}{
		{"status", true},
		{"priority", false},
		{"due_date", true},
		{"assignee", false},
		{"name", true},
	}
	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			loader := newFakeLoader()
			svc := app.NewUIStateServiceWithLoader(loader)

			require.NoError(t, svc.SetSortPreference("default", tc.field, tc.asc))

			gotField, gotAsc := svc.GetSortPreference("default")
			assert.Equal(t, tc.field, gotField)
			assert.Equal(t, tc.asc, gotAsc)
		})
	}
}

func TestSetSortPreference_ProfileNotFound_ReturnsError(t *testing.T) {
	loader := newFakeLoader()
	svc := app.NewUIStateServiceWithLoader(loader)

	err := svc.SetSortPreference("nonexistent", "status", true)
	require.Error(t, err)
	assert.ErrorIs(t, err, config.ErrProfileNotFound)
}

func TestSetSortPreference_SaveError_ReturnsError(t *testing.T) {
	loader := newFakeLoader()
	loader.saveErr = errors.New("write failed")
	svc := app.NewUIStateServiceWithLoader(loader)

	err := svc.SetSortPreference("default", "status", true)
	require.Error(t, err)
	assert.ErrorContains(t, err, "save config")
}

func TestSetSortPreference_LoadError_ReturnsError(t *testing.T) {
	loader := &fakeConfigLoader{loadErr: errors.New("disk error")}
	svc := app.NewUIStateServiceWithLoader(loader)

	err := svc.SetSortPreference("default", "status", true)
	require.Error(t, err)
	assert.ErrorContains(t, err, "load config")
}

// ── Profile isolation ─────────────────────────────────────────────────────────

func TestSetSortPreference_ProfileIsolation(t *testing.T) {
	// Two profiles: changes to one must not affect the other.
	cfg := config.New()
	cfg.SetProfile(&config.Profile{Name: "work", AuthMethod: config.AuthMethodPersonalToken})
	cfg.SetProfile(&config.Profile{Name: "personal", AuthMethod: config.AuthMethodPersonalToken})
	loader := &fakeConfigLoader{cfg: cfg}
	svc := app.NewUIStateServiceWithLoader(loader)

	require.NoError(t, svc.SetSortPreference("work", "due_date", true))
	require.NoError(t, svc.SetSortPreference("personal", "name", false))

	workField, workAsc := svc.GetSortPreference("work")
	assert.Equal(t, "due_date", workField)
	assert.True(t, workAsc)

	personalField, personalAsc := svc.GetSortPreference("personal")
	assert.Equal(t, "name", personalField)
	assert.False(t, personalAsc)
}

// ── Bookmark helpers ──────────────────────────────────────────────────────────

// newBookmark creates a test bookmark with a fixed timestamp.
func newBookmark(taskID, taskName, listID, listName string) config.Bookmark {
	return config.Bookmark{
		TaskID:   taskID,
		TaskName: taskName,
		ListID:   listID,
		ListName: listName,
		AddedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

// ── GetBookmarks ──────────────────────────────────────────────────────────────

func TestGetBookmarks_EmptyWhenNone(t *testing.T) {
	svc := app.NewUIStateServiceWithLoader(newFakeLoader())

	got := svc.GetBookmarks("default")

	assert.Nil(t, got, "GetBookmarks should return nil when no bookmarks exist")
}

func TestGetBookmarks_ProfileNotFound_ReturnsNil(t *testing.T) {
	svc := app.NewUIStateServiceWithLoader(newFakeLoader())

	got := svc.GetBookmarks("nonexistent")

	assert.Nil(t, got)
}

func TestGetBookmarks_LoadError_ReturnsNil(t *testing.T) {
	loader := &fakeConfigLoader{loadErr: errors.New("disk error")}
	svc := app.NewUIStateServiceWithLoader(loader)

	got := svc.GetBookmarks("default")

	assert.Nil(t, got)
}

func TestGetBookmarks_ReturnsCopy(t *testing.T) {
	loader := newFakeLoader()
	svc := app.NewUIStateServiceWithLoader(loader)
	b := newBookmark("t1", "Task 1", "l1", "List 1")
	require.NoError(t, svc.AddBookmark("default", b))

	got := svc.GetBookmarks("default")
	require.Len(t, got, 1)

	// Mutating the returned slice should not affect stored bookmarks.
	got[0].TaskName = "MUTATED"
	got2 := svc.GetBookmarks("default")
	assert.Equal(t, "Task 1", got2[0].TaskName, "GetBookmarks should return an independent copy")
}

// ── AddBookmark ───────────────────────────────────────────────────────────────

func TestAddBookmark_PersistsBookmark(t *testing.T) {
	loader := newFakeLoader()
	svc := app.NewUIStateServiceWithLoader(loader)
	b := newBookmark("t1", "Task 1", "l1", "List 1")

	err := svc.AddBookmark("default", b)
	require.NoError(t, err)

	got := svc.GetBookmarks("default")
	require.Len(t, got, 1)
	assert.Equal(t, "t1", got[0].TaskID)
	assert.Equal(t, "Task 1", got[0].TaskName)
	assert.Equal(t, "l1", got[0].ListID)
	assert.Equal(t, "List 1", got[0].ListName)
}

func TestAddBookmark_Idempotent(t *testing.T) {
	loader := newFakeLoader()
	svc := app.NewUIStateServiceWithLoader(loader)
	b := newBookmark("t1", "Task 1", "l1", "List 1")

	require.NoError(t, svc.AddBookmark("default", b))
	require.NoError(t, svc.AddBookmark("default", b)) // second add — same task ID

	got := svc.GetBookmarks("default")
	assert.Len(t, got, 1, "duplicate bookmark should not be added")
}

func TestAddBookmark_MultipleBookmarks(t *testing.T) {
	loader := newFakeLoader()
	svc := app.NewUIStateServiceWithLoader(loader)

	require.NoError(t, svc.AddBookmark("default", newBookmark("t1", "Task 1", "l1", "List 1")))
	require.NoError(t, svc.AddBookmark("default", newBookmark("t2", "Task 2", "l1", "List 1")))
	require.NoError(t, svc.AddBookmark("default", newBookmark("t3", "Task 3", "l2", "List 2")))

	got := svc.GetBookmarks("default")
	assert.Len(t, got, 3)
}

func TestAddBookmark_ProfileNotFound_ReturnsError(t *testing.T) {
	loader := newFakeLoader()
	svc := app.NewUIStateServiceWithLoader(loader)

	err := svc.AddBookmark("nonexistent", newBookmark("t1", "Task 1", "l1", "List 1"))
	require.Error(t, err)
	assert.ErrorIs(t, err, config.ErrProfileNotFound)
}

func TestAddBookmark_LoadError_ReturnsError(t *testing.T) {
	loader := &fakeConfigLoader{loadErr: errors.New("disk error")}
	svc := app.NewUIStateServiceWithLoader(loader)

	err := svc.AddBookmark("default", newBookmark("t1", "Task 1", "l1", "List 1"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "load config")
}

func TestAddBookmark_SaveError_ReturnsError(t *testing.T) {
	loader := newFakeLoader()
	loader.saveErr = errors.New("write failed")
	svc := app.NewUIStateServiceWithLoader(loader)

	err := svc.AddBookmark("default", newBookmark("t1", "Task 1", "l1", "List 1"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "save config")
}

// ── RemoveBookmark ────────────────────────────────────────────────────────────

func TestRemoveBookmark_RemovesExisting(t *testing.T) {
	loader := newFakeLoader()
	svc := app.NewUIStateServiceWithLoader(loader)
	require.NoError(t, svc.AddBookmark("default", newBookmark("t1", "Task 1", "l1", "List 1")))
	require.NoError(t, svc.AddBookmark("default", newBookmark("t2", "Task 2", "l1", "List 1")))

	err := svc.RemoveBookmark("default", "t1")
	require.NoError(t, err)

	got := svc.GetBookmarks("default")
	require.Len(t, got, 1)
	assert.Equal(t, "t2", got[0].TaskID)
}

func TestRemoveBookmark_NoOp_WhenNotFound(t *testing.T) {
	loader := newFakeLoader()
	svc := app.NewUIStateServiceWithLoader(loader)
	require.NoError(t, svc.AddBookmark("default", newBookmark("t1", "Task 1", "l1", "List 1")))

	err := svc.RemoveBookmark("default", "nonexistent")
	require.NoError(t, err)

	got := svc.GetBookmarks("default")
	assert.Len(t, got, 1, "remove of non-existent bookmark should be a no-op")
}

func TestRemoveBookmark_ProfileNotFound_ReturnsError(t *testing.T) {
	svc := app.NewUIStateServiceWithLoader(newFakeLoader())

	err := svc.RemoveBookmark("nonexistent", "t1")
	require.Error(t, err)
	assert.ErrorIs(t, err, config.ErrProfileNotFound)
}

func TestRemoveBookmark_LoadError_ReturnsError(t *testing.T) {
	loader := &fakeConfigLoader{loadErr: errors.New("disk error")}
	svc := app.NewUIStateServiceWithLoader(loader)

	err := svc.RemoveBookmark("default", "t1")
	require.Error(t, err)
	assert.ErrorContains(t, err, "load config")
}

// ── IsBookmarked ──────────────────────────────────────────────────────────────

func TestIsBookmarked_TrueWhenPresent(t *testing.T) {
	loader := newFakeLoader()
	svc := app.NewUIStateServiceWithLoader(loader)
	require.NoError(t, svc.AddBookmark("default", newBookmark("t1", "Task 1", "l1", "List 1")))

	assert.True(t, svc.IsBookmarked("default", "t1"))
}

func TestIsBookmarked_FalseWhenAbsent(t *testing.T) {
	svc := app.NewUIStateServiceWithLoader(newFakeLoader())

	assert.False(t, svc.IsBookmarked("default", "t1"))
}

func TestIsBookmarked_FalseAfterRemove(t *testing.T) {
	loader := newFakeLoader()
	svc := app.NewUIStateServiceWithLoader(loader)
	require.NoError(t, svc.AddBookmark("default", newBookmark("t1", "Task 1", "l1", "List 1")))
	require.NoError(t, svc.RemoveBookmark("default", "t1"))

	assert.False(t, svc.IsBookmarked("default", "t1"))
}

func TestIsBookmarked_ProfileNotFound_ReturnsFalse(t *testing.T) {
	svc := app.NewUIStateServiceWithLoader(newFakeLoader())

	assert.False(t, svc.IsBookmarked("nonexistent", "t1"))
}

func TestIsBookmarked_LoadError_ReturnsFalse(t *testing.T) {
	loader := &fakeConfigLoader{loadErr: errors.New("disk error")}
	svc := app.NewUIStateServiceWithLoader(loader)

	assert.False(t, svc.IsBookmarked("default", "t1"))
}

// ── Profile isolation ─────────────────────────────────────────────────────────

func TestBookmarks_ProfileIsolation(t *testing.T) {
	cfg := config.New()
	cfg.SetProfile(&config.Profile{Name: "work", AuthMethod: config.AuthMethodPersonalToken})
	cfg.SetProfile(&config.Profile{Name: "personal", AuthMethod: config.AuthMethodPersonalToken})
	loader := &fakeConfigLoader{cfg: cfg}
	svc := app.NewUIStateServiceWithLoader(loader)

	require.NoError(t, svc.AddBookmark("work", newBookmark("t1", "Work Task", "l1", "Work List")))
	require.NoError(t, svc.AddBookmark("personal", newBookmark("t2", "Personal Task", "l2", "Personal List")))

	workBMs := svc.GetBookmarks("work")
	assert.Len(t, workBMs, 1)
	assert.Equal(t, "t1", workBMs[0].TaskID)

	personalBMs := svc.GetBookmarks("personal")
	assert.Len(t, personalBMs, 1)
	assert.Equal(t, "t2", personalBMs[0].TaskID)

	assert.True(t, svc.IsBookmarked("work", "t1"))
	assert.False(t, svc.IsBookmarked("personal", "t1"))
}
