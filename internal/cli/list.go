package cli

import (
	"sort"

	"github.com/spf13/cobra"

	"github.com/meigma/whzbox/internal/ui"
)

func newListCommand(app **App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List cached sandboxes",
		Long: "Show every sandbox in the local cache, one row per provider kind.\n\n" +
			"This is a read-only view of the state file — it does not talk to\n" +
			"Whizlabs or refresh session tokens.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			sbs, err := (*app).Sandbox.List(cmd.Context())
			if err != nil {
				return err
			}
			sort.Slice(sbs, func(i, j int) bool {
				return sbs[i].Kind < sbs[j].Kind
			})
			if (*app).Config.JSON {
				return ui.RenderSandboxListJSON(cmd.OutOrStdout(), sbs)
			}
			ui.RenderSandboxList(cmd.OutOrStdout(), sbs, (*app).Clock.Now())
			return nil
		},
	}
}
