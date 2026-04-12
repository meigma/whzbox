package cli

import (
	"github.com/spf13/cobra"

	"github.com/meigma/whzbox/internal/ui"
)

func newStatusCommand(app **App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the cached session",
		Long: "Print the cached session (if any).\n\n" +
			"This command is read-only — it does not refresh tokens or prompt\n" +
			"for credentials, so it can safely be scripted.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			tokens, found, err := (*app).Session.Current(cmd.Context())
			if err != nil {
				return err
			}
			ui.RenderStatus(cmd.OutOrStdout(), tokens, found)
			return nil
		},
	}
}
