// Package auth — OAuth provider stub.
//
// This file holds a placeholder that satisfies the Provider interface for the
// OAuth method.  It intentionally returns an error at runtime so that callers
// discover the gap immediately.  Replace the body of Authorize (and add a token
// refresh mechanism) when the OAuth flow is implemented.
package auth

import (
	"context"
	"errors"
	"net/http"
)

// ErrOAuthNotImplemented is returned by OAuthProvider until the OAuth flow is
// fully implemented.
var ErrOAuthNotImplemented = errors.New("oauth authentication is not yet implemented")

// OAuthProvider is a future Provider implementation for OAuth 2.0.
// It exists to demonstrate the extension point and to confirm that the
// Provider interface supports OAuth without any changes.
type OAuthProvider struct{}

// Compile-time assertion that *OAuthProvider satisfies Provider.
var _ Provider = (*OAuthProvider)(nil)

// Method returns MethodOAuth.
func (o *OAuthProvider) Method() Method { return MethodOAuth }

// Authorize always returns ErrOAuthNotImplemented until this provider is wired
// to a live OAuth token source.
func (o *OAuthProvider) Authorize(_ context.Context, _ *http.Request) error {
	return ErrOAuthNotImplemented
}
