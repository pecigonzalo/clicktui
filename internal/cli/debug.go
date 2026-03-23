// Package cli — debug sub-command (non-interactive API testing).
package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pecigonzalo/clicktui/internal/app"
	"github.com/pecigonzalo/clicktui/internal/auth"
	"github.com/pecigonzalo/clicktui/internal/clickup"
)

func newDebugCmd() *cobra.Command {
	var workspaceFlag, spaceFlag string

	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Test API calls without the TUI",
		Long: `Run ClickUp API calls and print results to stdout.

Use this command to verify that authentication and API calls work without
launching the terminal UI.

Examples:
  clicktui debug                                     # list workspaces
  clicktui debug --workspace 2566449                 # list spaces in workspace
  clicktui debug --workspace 2566449 --space 901234  # list space contents`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			store := auth.NewKeyringStore()
			provider := auth.NewPersonalTokenProvider(profileFlag, store)
			client := clickup.New(provider)
			hierarchySvc := app.NewHierarchyService(client)
			ctx := cmd.Context()
			w := cmd.OutOrStdout()

			// Always load and print workspaces.
			_, _ = fmt.Fprintln(w, "=== Workspaces ===")
			workspaces, err := hierarchySvc.LoadWorkspaces(ctx)
			if err != nil {
				return fmt.Errorf("load workspaces: %w", err)
			}
			for _, ws := range workspaces {
				_, _ = fmt.Fprintf(w, "  [%s] %s  (%s)\n", ws.Kind, ws.Name, ws.ID)
			}

			if workspaceFlag == "" {
				return nil
			}

			// Load and print spaces for the given workspace.
			_, _ = fmt.Fprintf(w, "\n=== Spaces (workspace %s) ===\n", workspaceFlag)
			spaces, err := hierarchySvc.LoadSpaces(ctx, workspaceFlag)
			if err != nil {
				return fmt.Errorf("load spaces: %w", err)
			}
			for _, sp := range spaces {
				_, _ = fmt.Fprintf(w, "  [%s] %s  (%s)\n", sp.Kind, sp.Name, sp.ID)
			}

			if spaceFlag == "" {
				return nil
			}

			// Load and print space contents (folders + lists).
			_, _ = fmt.Fprintf(w, "\n=== Space Contents (space %s) ===\n", spaceFlag)
			contents, err := hierarchySvc.LoadSpaceContents(ctx, spaceFlag)
			if err != nil {
				return fmt.Errorf("load space contents: %w", err)
			}
			printNodes(w, contents, 0)

			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceFlag, "workspace", "", "workspace (team) ID")
	cmd.Flags().StringVar(&spaceFlag, "space", "", "space ID (requires --workspace)")

	return cmd
}

// printNodes recursively prints hierarchy nodes with indentation.
func printNodes(w io.Writer, nodes []*app.HierarchyNode, depth int) {
	indent := strings.Repeat("  ", depth+1)
	for _, n := range nodes {
		_, _ = fmt.Fprintf(w, "%s[%s] %s  (%s)\n", indent, n.Kind, n.Name, n.ID)
		if len(n.Children) > 0 {
			printNodes(w, n.Children, depth+1)
		}
	}
}
