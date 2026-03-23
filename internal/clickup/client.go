// Package clickup provides a minimal ClickUp API v2 client.
//
// The client depends only on the auth.Provider interface, which keeps it
// decoupled from the concrete authentication mechanism.  Swap the provider to
// switch auth methods without touching the client.
package clickup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pecigonzalo/clicktui/internal/auth"
)

const (
	defaultBaseURL = "https://api.clickup.com/api/v2"
	defaultTimeout = 30 * time.Second
)

// Client is a ClickUp API v2 client.  Construct one with New; do not use the
// zero value.
type Client struct {
	baseURL    string
	httpClient *http.Client
	provider   auth.Provider
}

// New returns a Client that authenticates using provider.
func New(provider auth.Provider) *Client {
	return &Client{
		baseURL:  defaultBaseURL,
		provider: provider,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// SetBaseURL overrides the API base URL.  Intended for tests only.
func (c *Client) SetBaseURL(u string) { c.baseURL = u }

// do executes an authenticated GET request and JSON-decodes the response body
// into out.  It returns an *APIError for non-2xx responses.
func (c *Client) do(ctx context.Context, method, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if err := c.provider.Authorize(ctx, req); err != nil {
		return fmt.Errorf("authorize request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http %s %s: %w", method, path, err)
	}
	defer func() {
		_ = resp.Body.Close() // body is fully read before return; close error is not actionable
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}
