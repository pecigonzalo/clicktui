// Package clickup — list member endpoints.
package clickup

import (
	"context"
	"fmt"
)

type membersResponse struct {
	Members []Member `json:"members"`
}

// ListMembers returns the members of a ClickUp list.
func (c *Client) ListMembers(ctx context.Context, listID string) ([]Member, error) {
	var out membersResponse
	if err := c.do(ctx, "GET", fmt.Sprintf("/list/%s/member", listID), &out); err != nil {
		return nil, err
	}
	return out.Members, nil
}
