package cli

import (
	"errors"
	"fmt"
	"os"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"

	"github.com/meigma/whzbox/internal/core/session"
	"github.com/meigma/whzbox/internal/ui"
)

func newDestroyCommand(app **App) *cobra.Command {
	return &cobra.Command{
		Use:   "destroy",
		Short: "Destroy the active sandbox",
		Long: "Tear down the user's currently active sandbox. All resources\n" +
			"inside it will be permanently deleted.\n\n" +
			"An interactive confirmation prompt is shown unless --yes is set.\n" +
			"In non-interactive environments --yes is required.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !(*app).Config.AssumeYes {
				confirmed, err := confirmDestroy()
				if err != nil {
					return err
				}
				if !confirmed {
					// User dismissed without confirming. Not an error:
					// treat as a successful "never mind".
					return nil
				}
			}
			return (*app).Sandbox.Destroy(cmd.Context())
		},
	}
}

// confirmDestroy shows an interactive huh confirm prompt. In a non-TTY
// environment it returns ErrPromptUnavailable so the CLI layer maps to
// exit 5 — scripts must explicitly pass --yes.
func confirmDestroy() (bool, error) {
	if !ui.IsInteractive(os.Stdin) || !ui.IsInteractive(os.Stderr) {
		return false, fmt.Errorf("%w: --yes is required for non-interactive destroy", session.ErrPromptUnavailable)
	}

	var confirmed bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Destroy the active sandbox?").
				Description("All resources inside the sandbox will be permanently deleted.").
				Affirmative("Yes, destroy").
				Negative("Cancel").
				Value(&confirmed),
		),
	).WithTheme(huh.ThemeFunc(ui.HuhTheme))

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return false, session.ErrUserAborted
		}
		return false, fmt.Errorf("destroy confirm: %w", err)
	}
	return confirmed, nil
}
