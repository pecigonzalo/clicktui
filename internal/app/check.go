// Package app — compile-time interface assertions.
package app

import "github.com/pecigonzalo/clicktui/internal/clickup"

// Verify that *clickup.Client satisfies the ClickUpAPI interface at compile time.
var _ ClickUpAPI = (*clickup.Client)(nil)
