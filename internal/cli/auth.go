// Package cli — auth sub-commands (login, logout, status).
package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pecigonzalo/clicktui/internal/auth"
	"github.com/pecigonzalo/clicktui/internal/clickup"
	"github.com/pecigonzalo/clicktui/internal/config"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
	}
	cmd.AddCommand(newAuthLoginCmd(), newAuthLogoutCmd(), newAuthStatusCmd())
	return cmd
}

// newAuthLoginCmd stores a personal token for the active profile.
func newAuthLoginCmd() *cobra.Command {
	var token string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Save credentials for the active profile",
		Long: `Save credentials for the selected profile.

Currently only personal API tokens are supported.  Run 'clicktui auth login --token <token>'
to store your ClickUp personal API token in the OS keyring.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if token == "" {
				return fmt.Errorf("--token is required")
			}

			store := auth.NewKeyringStore()
			if err := store.Set(profileFlag, token); err != nil {
				return fmt.Errorf("save credentials: %w", err)
			}

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			profile := &config.Profile{
				Name:       profileFlag,
				AuthMethod: config.AuthMethodPersonalToken,
			}

			// Preserve existing workspace/space IDs if the profile already exists.
			if existing, err := cfg.Profile(profileFlag); err == nil {
				profile.WorkspaceID = existing.WorkspaceID
				profile.SpaceID = existing.SpaceID
			}

			// Auto-detect workspace when none is configured and exactly one exists.
			if profile.WorkspaceID == "" {
				if wsID, err := detectSingleWorkspace(cmd.Context(), token); err == nil && wsID != "" {
					profile.WorkspaceID = wsID
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Auto-detected workspace %s.\n", wsID)
				}
			}

			cfg.SetProfile(profile)
			cfg.ActiveProfile = profileFlag
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Credentials saved for profile %q.\n", profileFlag)
			return err
		},
	}

	cmd.Flags().StringVarP(&token, "token", "t", "", "ClickUp personal API token")
	return cmd
}

// detectSingleWorkspace fetches workspaces for the given token and returns the
// workspace ID if exactly one exists. Returns ("", nil) when detection is not
// possible or multiple workspaces exist.
func detectSingleWorkspace(ctx context.Context, token string) (string, error) {
	provider := auth.NewStaticTokenProvider(token)
	client := clickup.New(provider)
	teams, err := client.Teams(ctx)
	if err != nil {
		return "", err
	}
	if len(teams) == 1 {
		return teams[0].ID, nil
	}
	return "", nil
}

// newAuthLogoutCmd removes credentials for the active profile.
func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove credentials for the active profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			store := auth.NewKeyringStore()
			if err := store.Delete(profileFlag); err != nil {
				return fmt.Errorf("remove credentials: %w", err)
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "Credentials removed for profile %q.\n", profileFlag)
			return err
		},
	}
}

// newAuthStatusCmd verifies the stored credential against the ClickUp API.
func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check authentication status for the active profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			store := auth.NewKeyringStore()
			provider := auth.NewPersonalTokenProvider(profileFlag, store)
			client := clickup.New(provider)

			user, err := client.AuthorizedUser(context.Background())
			if err != nil {
				var apiErr *clickup.APIError
				if errors.As(err, &apiErr) && apiErr.StatusCode == 401 {
					return fmt.Errorf("not authenticated: invalid or missing token for profile %q", profileFlag)
				}
				return fmt.Errorf("auth check: %w", err)
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(),
				"Authenticated as %s (%s) using profile %q.\n",
				user.Username, user.Email, profileFlag,
			)
			return err
		},
	}
}
