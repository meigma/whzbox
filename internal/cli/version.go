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

// BuildInfo is the immutable build metadata exposed by a command tree.
type BuildInfo struct {
	Version string
	Commit  string
	Time    string
}

// CurrentBuildInfo snapshots the link-time build variables.
func CurrentBuildInfo() BuildInfo {
	return BuildInfo{Version: Version, Commit: Commit, Time: BuildTime}
}

// String formats build metadata for user display.
func (b BuildInfo) String() string {
	return fmt.Sprintf("%s (%s) built %s", b.Version, b.Commit, b.Time)
}

// BuildString formats the build metadata for user display.
func BuildString() string {
	return CurrentBuildInfo().String()
}

func newVersionCommand(app **App) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Args:  cobra.NoArgs,
		Short: "Print version information",
		Long:  "Print the version, commit, and build timestamp of this whzbox binary.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			build := CurrentBuildInfo()
			if app != nil && *app != nil {
				build = (*app).Build
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "whzbox %s\n", build.String()); err != nil {
				return err
			}
			if app != nil && *app != nil && (*app).Logger != nil {
				(*app).Logger.Debug("version command invoked",
					"version", build.Version,
					"commit", build.Commit,
					"build_time", build.Time,
				)
			}
			return nil
		},
	}
}
