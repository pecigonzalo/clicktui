// Package cli — config sub-commands (schema).
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/pecigonzalo/clicktui/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}
	cmd.AddCommand(newConfigSchemaCmd())
	return cmd
}

// newConfigSchemaCmd writes the JSON schema file to the config directory.
func newConfigSchemaCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "schema",
		Short: "Install editor schema hints for config file",
		Long: `Write the JSON Schema file to the config directory.

This creates a config.schema.json file that editors with the YAML Language Server
extension can use to provide inline validation and completion for your config file.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			schemaPath, err := config.WriteSchema()
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Schema written to %s\n", schemaPath)

			// Check if config.yaml exists and if it has the yaml-language-server comment.
			configPath, err := config.ConfigFilePath()
			if err == nil {
				if data, err := os.ReadFile(configPath); err == nil {
					content := string(data)
					if !containsYAMLLSPComment(content) {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(),
							"To enable editor hints, add this comment to the top of %s:\n",
							filepath.Base(configPath),
						)
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s", "# yaml-language-server: $schema=config.schema.json\n")
					}
				}
			}

			return nil
		},
	}
}

// containsYAMLLSPComment checks if content already has the yaml-language-server directive.
func containsYAMLLSPComment(content string) bool {
	return len(content) >= 29 && content[0:29] == "# yaml-language-server: "
}
