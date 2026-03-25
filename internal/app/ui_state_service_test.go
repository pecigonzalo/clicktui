package app_test

import (
	"errors"
	"testing"

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
	assert.Contains(t, err.Error(), "save config")
}

func TestSetSortPreference_LoadError_ReturnsError(t *testing.T) {
	loader := &fakeConfigLoader{loadErr: errors.New("disk error")}
	svc := app.NewUIStateServiceWithLoader(loader)

	err := svc.SetSortPreference("default", "status", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
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
