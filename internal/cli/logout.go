package cli

import (
	"github.com/spf13/cobra"
)

func newLogoutCommand(app **App) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear the cached session",
		Long: "Remove the cached Whizlabs tokens from disk. Subsequent commands\n" +
			"will prompt for credentials again.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return (*app).Session.Logout(cmd.Context())
		},
	}
}
