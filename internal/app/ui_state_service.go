package app

import (
	"fmt"
	"sync"

	"github.com/pecigonzalo/clicktui/internal/config"
)

// ConfigLoader abstracts config loading for UIStateService.
type ConfigLoader interface {
	Load() (*config.Config, error)
	Save(cfg *config.Config) error
}

// defaultConfigLoader delegates to the package-level config functions.
type defaultConfigLoader struct{}

func (defaultConfigLoader) Load() (*config.Config, error) { return config.Load() }
func (defaultConfigLoader) Save(cfg *config.Config) error { return config.Save(cfg) }

// UIStateService loads and persists per-profile UI state (e.g. sort preferences).
// It is safe for concurrent use.
type UIStateService struct {
	mu     sync.Mutex
	loader ConfigLoader
}

// NewUIStateService creates a UIStateService that persists state via the
// package-level config.Load / config.Save functions.
func NewUIStateService() *UIStateService {
	return &UIStateService{loader: defaultConfigLoader{}}
}

// NewUIStateServiceWithLoader creates a UIStateService backed by the given
// loader. Primarily intended for testing; production code should use
// NewUIStateService.
func NewUIStateServiceWithLoader(loader ConfigLoader) *UIStateService {
	return &UIStateService{loader: loader}
}

// GetSortPreference returns the persisted sort field and direction for the
// named profile. When no preference has been saved, it returns ("", false).
func (s *UIStateService) GetSortPreference(profile string) (field string, ascending bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.loader.Load()
	if err != nil {
		return "", false
	}
	p, err := cfg.Profile(profile)
	if err != nil {
		return "", false
	}
	return p.UIState.SortField, p.UIState.SortAsc
}

// SetSortPreference persists the sort field and direction for the named profile.
// It returns an error when the profile does not exist or the config cannot be saved.
func (s *UIStateService) SetSortPreference(profile string, field string, ascending bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.loader.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	p, err := cfg.Profile(profile)
	if err != nil {
		return fmt.Errorf("set sort preference: %w", err)
	}
	p.UIState.SortField = field
	p.UIState.SortAsc = ascending
	if err := s.loader.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}

// GetBookmarks returns a copy of the bookmark list for the named profile.
// Returns nil when the profile does not exist or cannot be loaded.
func (s *UIStateService) GetBookmarks(profile string) []config.Bookmark {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.loader.Load()
	if err != nil {
		return nil
	}
	p, err := cfg.Profile(profile)
	if err != nil {
		return nil
	}
	if len(p.UIState.Bookmarks) == 0 {
		return nil
	}
	out := make([]config.Bookmark, len(p.UIState.Bookmarks))
	copy(out, p.UIState.Bookmarks)
	return out
}

// AddBookmark appends b to the bookmark list for the named profile, unless a
// bookmark for b.TaskID already exists (idempotent add). Returns an error when
// the profile does not exist or the config cannot be saved.
func (s *UIStateService) AddBookmark(profile string, b config.Bookmark) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.loader.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	p, err := cfg.Profile(profile)
	if err != nil {
		return fmt.Errorf("add bookmark: %w", err)
	}
	// Idempotent: skip if already bookmarked.
	for _, existing := range p.UIState.Bookmarks {
		if existing.TaskID == b.TaskID {
			return nil
		}
	}
	p.UIState.Bookmarks = append(p.UIState.Bookmarks, b)
	if err := s.loader.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}

// RemoveBookmark removes the bookmark for taskID from the named profile.
// It is a no-op when no bookmark with that taskID exists. Returns an error when
// the profile does not exist or the config cannot be saved.
func (s *UIStateService) RemoveBookmark(profile string, taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.loader.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	p, err := cfg.Profile(profile)
	if err != nil {
		return fmt.Errorf("remove bookmark: %w", err)
	}
	filtered := p.UIState.Bookmarks[:0]
	for _, b := range p.UIState.Bookmarks {
		if b.TaskID != taskID {
			filtered = append(filtered, b)
		}
	}
	// Only save if something actually changed.
	if len(filtered) == len(p.UIState.Bookmarks) {
		return nil
	}
	p.UIState.Bookmarks = filtered
	if err := s.loader.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}

// IsBookmarked reports whether a bookmark exists for taskID in the named profile.
// Returns false when the profile cannot be loaded.
func (s *UIStateService) IsBookmarked(profile string, taskID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.loader.Load()
	if err != nil {
		return false
	}
	p, err := cfg.Profile(profile)
	if err != nil {
		return false
	}
	for _, b := range p.UIState.Bookmarks {
		if b.TaskID == taskID {
			return true
		}
	}
	return false
}
