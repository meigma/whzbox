package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/meigma/whzbox/internal/core/sandbox"
)

func newMFACommand(app **App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mfa <provider>",
		Short: "Generate a console MFA code",
		Long: "Generate the current short-lived MFA code for an active Azure sandbox.\n" +
			"At the Microsoft MFA prompt, select \"Use a verification code\", then enter this code.\n" +
			"The code is printed to stdout and is never cached.",
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{string(sandbox.KindAzure)},
		RunE: func(cmd *cobra.Command, args []string) error {
			code, err := (*app).Sandbox.GenerateMFA(cmd.Context(), sandbox.Kind(args[0]))
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), code)
			return err
		},
	}
	return cmd
}
