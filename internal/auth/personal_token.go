// Package auth — personal-token provider.
package auth

import (
	"context"
	"fmt"
	"net/http"
)

// PersonalTokenProvider authenticates HTTP requests using a ClickUp personal
// API token.
//
// ClickUp's v2 API expects the token directly in the Authorization header
// (without a "Bearer" prefix).  This implementation hides that detail from
// callers; if the header format changes or OAuth is added, only this provider
// needs updating.
type PersonalTokenProvider struct {
	profile string
	store   CredentialStore
}

// Compile-time assertion that *PersonalTokenProvider satisfies Provider.
var _ Provider = (*PersonalTokenProvider)(nil)

// NewPersonalTokenProvider returns a Provider that reads the personal token for
// profile from store on each call to Authorize.
func NewPersonalTokenProvider(profile string, store CredentialStore) *PersonalTokenProvider {
	return &PersonalTokenProvider{profile: profile, store: store}
}

// Method returns MethodPersonalToken.
func (p *PersonalTokenProvider) Method() Method { return MethodPersonalToken }

// Authorize injects the Authorization header with the stored personal token.
// It returns an error if the token cannot be retrieved.
func (p *PersonalTokenProvider) Authorize(_ context.Context, r *http.Request) error {
	token, err := p.store.Get(p.profile)
	if err != nil {
		return fmt.Errorf("authorize: %w", err)
	}
	r.Header.Set("Authorization", token)
	return nil
}
