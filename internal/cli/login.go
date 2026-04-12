package cli

import (
	"github.com/spf13/cobra"
)

func newLoginCommand(app **App) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Sign in to Whizlabs",
		Long: "Sign in to Whizlabs interactively. The resulting session is saved to\n" +
			"disk and reused by subsequent commands until the refresh token expires.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := (*app).Session.Login(cmd.Context())
			return err
		},
	}
}
