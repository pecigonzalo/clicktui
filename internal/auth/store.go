// Package auth — keyring-backed credential store.
package auth

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

const keyringSvc = "clicktui"

// ErrNotFound is returned by CredentialStore.Get when no credential exists.
var ErrNotFound = errors.New("credential not found")

// KeyringStore is a CredentialStore backed by the OS keyring (Keychain on macOS,
// Secret Service on Linux, Windows Credential Manager on Windows).
//
// Credentials are stored under a fixed service name and the profile as the
// account key so multiple profiles are naturally namespaced.
type KeyringStore struct{}

// Compile-time assertion that *KeyringStore satisfies CredentialStore.
var _ CredentialStore = (*KeyringStore)(nil)

// NewKeyringStore returns a KeyringStore ready for use.
func NewKeyringStore() *KeyringStore { return &KeyringStore{} }

// Get retrieves the credential for profile.
func (s *KeyringStore) Get(profile string) (string, error) {
	secret, err := keyring.Get(keyringSvc, profile)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", fmt.Errorf("%w: profile %q", ErrNotFound, profile)
	}
	if err != nil {
		return "", fmt.Errorf("keyring get: %w", err)
	}
	return secret, nil
}

// Set persists credential for profile in the OS keyring.
func (s *KeyringStore) Set(profile string, credential string) error {
	if err := keyring.Set(keyringSvc, profile, credential); err != nil {
		return fmt.Errorf("keyring set: %w", err)
	}
	return nil
}

// Delete removes the credential for profile from the OS keyring.
func (s *KeyringStore) Delete(profile string) error {
	if err := keyring.Delete(keyringSvc, profile); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("keyring delete: %w", err)
	}
	return nil
}
