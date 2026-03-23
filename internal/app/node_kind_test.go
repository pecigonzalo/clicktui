package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pecigonzalo/clicktui/internal/app"
)

func TestNodeKind_String(t *testing.T) {
	tests := []struct {
		kind app.NodeKind
		want string
	}{
		{app.NodeWorkspace, "Workspace"},
		{app.NodeSpace, "Space"},
		{app.NodeFolder, "Folder"},
		{app.NodeList, "List"},
		{app.NodeKind(99), "Unknown"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.kind.String())
	}
}
