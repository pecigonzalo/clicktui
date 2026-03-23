// Package clickup — space, folder, and list hierarchy endpoints.
package clickup

import (
	"context"
	"fmt"
)

type spacesResponse struct {
	Spaces []Space `json:"spaces"`
}

type foldersResponse struct {
	Folders []Folder `json:"folders"`
}

type listsResponse struct {
	Lists []List `json:"lists"`
}

// Spaces returns all spaces in the given workspace (team).
func (c *Client) Spaces(ctx context.Context, teamID string) ([]Space, error) {
	var out spacesResponse
	if err := c.do(ctx, "GET", fmt.Sprintf("/team/%s/space", teamID), &out); err != nil {
		return nil, err
	}
	return out.Spaces, nil
}

// Folders returns all folders in the given space.
func (c *Client) Folders(ctx context.Context, spaceID string) ([]Folder, error) {
	var out foldersResponse
	if err := c.do(ctx, "GET", fmt.Sprintf("/space/%s/folder", spaceID), &out); err != nil {
		return nil, err
	}
	return out.Folders, nil
}

// FolderlessLists returns lists that are not inside any folder for the given space.
func (c *Client) FolderlessLists(ctx context.Context, spaceID string) ([]List, error) {
	var out listsResponse
	if err := c.do(ctx, "GET", fmt.Sprintf("/space/%s/list", spaceID), &out); err != nil {
		return nil, err
	}
	return out.Lists, nil
}
