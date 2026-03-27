package auth_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pecigonzalo/clicktui/internal/auth"
)

// stubStore is an in-memory CredentialStore for tests.
type stubStore struct {
	creds map[string]string
}

var _ auth.CredentialStore = (*stubStore)(nil)

func newStubStore() *stubStore {
	return &stubStore{creds: make(map[string]string)}
}

func (s *stubStore) Get(profile string) (string, error) {
	v, ok := s.creds[profile]
	if !ok {
		return "", auth.ErrNotFound
	}
	return v, nil
}

func (s *stubStore) Set(profile, cred string) error {
	s.creds[profile] = cred
	return nil
}

func (s *stubStore) Delete(profile string) error {
	delete(s.creds, profile)
	return nil
}

func TestPersonalTokenProvider_Method(t *testing.T) {
	p := auth.NewPersonalTokenProvider("default", newStubStore())
	assert.Equal(t, auth.MethodPersonalToken, p.Method())
}

func TestPersonalTokenProvider_Authorize_SetsHeader(t *testing.T) {
	store := newStubStore()
	require.NoError(t, store.Set("default", "pk_test_abc123"))

	p := auth.NewPersonalTokenProvider("default", store)
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)

	require.NoError(t, p.Authorize(context.Background(), req))
	assert.Equal(t, "pk_test_abc123", req.Header.Get("Authorization"))
}

func TestPersonalTokenProvider_Authorize_MissingToken(t *testing.T) {
	p := auth.NewPersonalTokenProvider("default", newStubStore())
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)

	err := p.Authorize(context.Background(), req)
	require.Error(t, err)
}

func TestOAuthProvider_Method(t *testing.T) {
	o := &auth.OAuthProvider{}
	assert.Equal(t, auth.MethodOAuth, o.Method())
}

func TestOAuthProvider_Authorize_ReturnsNotImplemented(t *testing.T) {
	o := &auth.OAuthProvider{}
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	err := o.Authorize(context.Background(), req)
	assert.ErrorIs(t, err, auth.ErrOAuthNotImplemented)
}

func TestStaticTokenProvider_Method(t *testing.T) {
	p := auth.NewStaticTokenProvider("pk_test_static")
	assert.Equal(t, auth.MethodPersonalToken, p.Method())
}

func TestStaticTokenProvider_Authorize(t *testing.T) {
	p := auth.NewStaticTokenProvider("pk_test_static")
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)

	require.NoError(t, p.Authorize(context.Background(), req))
	assert.Equal(t, "pk_test_static", req.Header.Get("Authorization"))
}
