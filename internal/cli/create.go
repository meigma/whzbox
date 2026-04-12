package cli

import (
	"errors"
	"time"

	"github.com/spf13/cobra"

	"github.com/meigma/whzbox/internal/core/sandbox"
	"github.com/meigma/whzbox/internal/ui"
)

func newCreateCommand(app **App) *cobra.Command {
	var duration time.Duration

	cmd := &cobra.Command{
		Use:   "create <provider>",
		Short: "Create a new sandbox",
		Long: "Create a new cloud sandbox through Whizlabs and render its\n" +
			"credentials.\n\n" +
			"v1 supports only the 'aws' provider. Duration must be between 1h\n" +
			"and 9h.",
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{"aws"},
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := sandbox.Kind(args[0])

			sb, err := (*app).Sandbox.Create(cmd.Context(), kind, duration)
			if err != nil {
				if errors.Is(err, sandbox.ErrVerificationFailed) && sb != nil {
					ui.RenderSandbox(cmd.OutOrStdout(), sb)
					(*app).Logger.Warn("credentials not verified; use with caution", "err", err)
				}
				return err
			}
			ui.RenderSandbox(cmd.OutOrStdout(), sb)
			return nil
		},
	}
	cmd.Flags().DurationVar(&duration, "duration", time.Hour, "sandbox lifetime (1h-9h)")
	return cmd
}
