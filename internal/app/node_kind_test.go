package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pecigonzalo/clicktui/internal/app"
)

func TestNodeKind_String(t *testing.T) {
	tests := []struct {
		name string
		kind app.NodeKind
		want string
	}{
		{"workspace", app.NodeWorkspace, "Workspace"},
		{"space", app.NodeSpace, "Space"},
		{"folder", app.NodeFolder, "Folder"},
		{"list", app.NodeList, "List"},
		{"unknown", app.NodeKind(99), "Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.kind.String())
		})
	}
}
