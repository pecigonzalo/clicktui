// Package clickup — workspace (team) and authenticated user endpoints.
package clickup

import "context"

type authorizedUserResponse struct {
	User User `json:"user"`
}

type teamsResponse struct {
	Teams []Team `json:"teams"`
}

// AuthorizedUser returns the user associated with the current credential.
// This is the canonical "am I authenticated?" check.
func (c *Client) AuthorizedUser(ctx context.Context) (*User, error) {
	var out authorizedUserResponse
	if err := c.do(ctx, "GET", "/user", &out); err != nil {
		return nil, err
	}
	return &out.User, nil
}

// Teams returns all workspaces (teams) accessible to the current credential.
func (c *Client) Teams(ctx context.Context) ([]Team, error) {
	var out teamsResponse
	if err := c.do(ctx, "GET", "/team", &out); err != nil {
		return nil, err
	}
	return out.Teams, nil
}
