// Package app — flattened, searchable list index for the command palette.
//
// LoadAllLists crawls every workspace/space/folder the token has access to
// and flattens the result into ListRef entries so the TUI can offer a single
// fuzzy-searchable "jump to any list" command, independent of what the
// hierarchy tree happens to have expanded so far.
package app

import (
	"context"
	"fmt"
	"sort"

	"github.com/sahilm/fuzzy"
)

// ListRef is a flattened reference to a single list, carrying enough context
// to render a breadcrumb (workspace › space › folder › list) in search results.
type ListRef struct {
	ID            string
	Name          string
	WorkspaceID   string
	WorkspaceName string
	SpaceName     string
	FolderName    string // "" for folderless lists
}

// LoadAllLists returns every list across every workspace the current token
// can see, flattened for search. The result is cached after the first
// successful load; call InvalidateAllLists to force a refresh.
func (s *HierarchyService) LoadAllLists(ctx context.Context) ([]ListRef, error) {
	s.allListsMu.Lock()
	if s.allListsLoaded {
		cached := s.allLists
		s.allListsMu.Unlock()
		return cached, nil
	}
	s.allListsMu.Unlock()

	v, err, _ := s.allListsGroup.Do("all", func() (any, error) {
		refs, err := s.crawlAllLists(ctx)
		if err != nil {
			return nil, err
		}
		s.allListsMu.Lock()
		s.allLists = refs
		s.allListsLoaded = true
		s.allListsMu.Unlock()
		return refs, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]ListRef), nil
}

// InvalidateAllLists evicts the cached list index so the next LoadAllLists
// call re-crawls the workspace hierarchy.
func (s *HierarchyService) InvalidateAllLists() {
	s.allListsMu.Lock()
	s.allListsLoaded = false
	s.allLists = nil
	s.allListsMu.Unlock()
	s.allListsGroup.Forget("all")
}

// crawlAllLists walks every workspace → space → (folder|folderless) and
// flattens all lists found into ListRef entries.
func (s *HierarchyService) crawlAllLists(ctx context.Context) ([]ListRef, error) {
	teams, err := s.api.Teams(ctx)
	if err != nil {
		return nil, fmt.Errorf("load workspaces: %w", err)
	}

	var refs []ListRef
	for _, team := range teams {
		spaces, err := s.api.Spaces(ctx, team.ID)
		if err != nil {
			return nil, fmt.Errorf("load spaces for workspace %s: %w", team.ID, err)
		}
		for _, space := range spaces {
			folders, err := s.api.Folders(ctx, space.ID)
			if err != nil {
				return nil, fmt.Errorf("load folders for space %s: %w", space.ID, err)
			}
			for _, f := range folders {
				for _, l := range f.Lists {
					refs = append(refs, ListRef{
						ID:            l.ID,
						Name:          l.Name,
						WorkspaceID:   team.ID,
						WorkspaceName: team.Name,
						SpaceName:     space.Name,
						FolderName:    f.Name,
					})
				}
			}

			folderlessLists, err := s.api.FolderlessLists(ctx, space.ID)
			if err != nil {
				return nil, fmt.Errorf("load folderless lists for space %s: %w", space.ID, err)
			}
			for _, l := range folderlessLists {
				refs = append(refs, ListRef{
					ID:            l.ID,
					Name:          l.Name,
					WorkspaceID:   team.ID,
					WorkspaceName: team.Name,
					SpaceName:     space.Name,
				})
			}
		}
	}
	return refs, nil
}

// FilterListRefs fuzzy-matches refs against query (matched on Name) and
// returns results ordered by match score, best first. An empty query returns
// refs unchanged (unlike the tree/task filters, there is no "unfiltered" nil
// sentinel here — callers always get a usable slice to render).
func FilterListRefs(refs []ListRef, query string) []ListRef {
	if query == "" {
		return refs
	}

	matches := fuzzy.FindFrom(query, listRefNames(refs))
	if len(matches) == 0 {
		return nil
	}

	sort.SliceStable(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	result := make([]ListRef, len(matches))
	for i, m := range matches {
		result[i] = refs[m.Index]
	}
	return result
}

// listRefNames adapts a ListRef slice for the fuzzy matching library.
type listRefNames []ListRef

func (n listRefNames) String(i int) string { return n[i].Name }
func (n listRefNames) Len() int            { return len(n) }
