package cli

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewRootCommand returns the root whzbox cobra command with persistent
// flags bound to a fresh Viper instance.
//
// Subcommands share a dependency container (*App) populated in
// PersistentPreRunE, so that command RunE funcs see a fully-initialised
// config, logger, and clock regardless of which command was invoked.
func NewRootCommand() *cobra.Command {
	vp := viper.New()
	var app *App

	cmd := &cobra.Command{
		Use:           "whzbox",
		Short:         "Spin up cloud sandboxes from Whizlabs",
		Long:          "whzbox spins up on-demand cloud sandboxes through Whizlabs, fetches their credentials, and verifies they work.",
		Version:       BuildString(),
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return initApp(vp, cmd, &app)
		},
	}

	// Persistent flags. Every value here is also reachable via the
	// WHZBOX_<UPPER> environment variable (AutomaticEnv with a - → _
	// key replacer).
	cmd.PersistentFlags().String("log-level", "", "log level (debug, info, warn, error)")
	cmd.PersistentFlags().CountP("verbose", "v", "increase verbosity (repeatable)")
	cmd.PersistentFlags().BoolP("quiet", "q", false, "suppress non-essential output")
	cmd.PersistentFlags().Bool("no-color", false, "disable colored output")
	cmd.PersistentFlags().Bool("yes", false, "skip confirmation prompts")
	cmd.PersistentFlags().Bool("json", false, "emit machine-readable JSON instead of styled output")

	cmd.AddCommand(
		newCreateCommand(&app),
		newDestroyCommand(&app),
		newStatusCommand(&app),
		newListCommand(&app),
		newExecCommand(&app),
		newLoginCommand(&app),
		newLogoutCommand(&app),
		newVersionCommand(&app),
	)

	return cmd
}

// initApp configures Viper (env prefix + flag binding) and constructs the
// shared *App container. It is called once per command execution from
// PersistentPreRunE.
func initApp(vp *viper.Viper, cmd *cobra.Command, app **App) error {
	vp.SetEnvPrefix("WHZBOX")
	vp.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	vp.AutomaticEnv()

	if err := vp.BindPFlags(cmd.Flags()); err != nil {
		return err
	}

	newApp := NewApp
	if isVersionCommand(cmd) {
		newApp = NewMetadataApp
	}

	a, err := newApp(vp)
	if err != nil {
		return err
	}
	*app = a
	return nil
}

func isVersionCommand(cmd *cobra.Command) bool {
	return cmd.Name() == "version"
}
