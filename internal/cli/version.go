package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Build metadata populated via -ldflags:
//
//	-X github.com/meigma/whzbox/internal/cli.Version=v1.2.3
//	-X github.com/meigma/whzbox/internal/cli.Commit=abc123
//	-X github.com/meigma/whzbox/internal/cli.BuildTime=2026-04-11T00:00:00Z
var (
	Version   = "dev"     //nolint:gochecknoglobals // populated via -ldflags at build time
	Commit    = "none"    //nolint:gochecknoglobals // populated via -ldflags at build time
	BuildTime = "unknown" //nolint:gochecknoglobals // populated via -ldflags at build time
)

// BuildString formats the build metadata for user display.
func BuildString() string {
	return fmt.Sprintf("%s (%s) built %s", Version, Commit, BuildTime)
}

func newVersionCommand(app **App) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Print the version, commit, and build timestamp of this whzbox binary.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "whzbox %s\n", BuildString()); err != nil {
				return err
			}
			// Demonstrate the logger is wired; debug levels will surface
			// this when the user runs with -vv or --log-level debug.
			if app != nil && *app != nil && (*app).Logger != nil {
				(*app).Logger.Debug("version command invoked",
					"version", Version,
					"commit", Commit,
					"build_time", BuildTime,
				)
			}
			return nil
		},
	}
}
