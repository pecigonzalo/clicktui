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
