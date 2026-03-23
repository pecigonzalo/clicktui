// Package auth defines the authentication abstraction used throughout the app.
//
// The central interfaces — Provider and CredentialStore — decouple the HTTP
// client and CLI commands from the concrete authentication mechanism.  Adding
// OAuth later requires only a new Provider implementation; no other layer needs
// to change.
package auth

import (
	"context"
	"net/http"
)

// Method identifies the kind of authentication a Provider implements.
type Method string

const (
	// MethodPersonalToken uses a ClickUp personal API token.
	MethodPersonalToken Method = "personal_token"
	// MethodOAuth uses OAuth 2.0 (not yet implemented).
	MethodOAuth Method = "oauth"
)

// Provider is the single point of contact between the HTTP client and the
// underlying credential source.  An implementation must be safe for concurrent
// use after construction.
type Provider interface {
	// Method reports which authentication mechanism this provider implements.
	Method() Method

	// Authorize injects the appropriate authorization header(s) into r.
	// It may return an error if the credential is missing or expired.
	Authorize(ctx context.Context, r *http.Request) error
}

// CredentialStore handles secure persistence and retrieval of credentials.
// Implementations are responsible for the storage medium (OS keychain, file,
// etc.) and must never log credential values.
type CredentialStore interface {
	// Get returns the credential stored under profile.  Returns ErrNotFound if
	// no credential exists for the given profile.
	Get(profile string) (string, error)

	// Set persists credential for profile, overwriting any prior value.
	Set(profile string, credential string) error

	// Delete removes the credential stored under profile.
	Delete(profile string) error
}
