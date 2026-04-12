package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/meigma/whzbox/internal/core/sandbox"
)

// validExecKinds is the fixed set of provider kinds accepted as the
// first positional arg to `whzbox exec`. Mirrors create's ValidArgs.
var validExecKinds = map[string]sandbox.Kind{ //nolint:gochecknoglobals // static registry
	"aws": sandbox.KindAWS,
}

func newExecCommand(app **App) *cobra.Command {
	var useShell bool

	cmd := &cobra.Command{
		Use:   "exec <provider> [-- cmd args...]",
		Short: "Run a command with the sandbox env for a provider",
		Long: "Run a command (or drop into $SHELL) with the cached sandbox's\n" +
			"credentials exported as environment variables.\n\n" +
			"  whzbox exec aws -- aws sts get-caller-identity\n" +
			"  whzbox exec aws -s \"aws s3 ls | head\"\n" +
			"  whzbox exec aws                     # drops into $SHELL\n\n" +
			"Fails if there is no active cached sandbox for the provider.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, ok := validExecKinds[args[0]]
			if !ok {
				return fmt.Errorf("%w: %q (want one of: aws)", sandbox.ErrUnknownKind, args[0])
			}

			sb, found, err := (*app).Sandbox.Load(cmd.Context(), kind)
			if err != nil {
				return err
			}
			if !found || !sb.ExpiresAt.After((*app).Clock.Now()) {
				return fmt.Errorf("no active %s sandbox — run: whzbox create %s", kind, kind)
			}

			rest := args[1:]
			if useShell && len(rest) != 1 {
				return fmt.Errorf("--shell takes exactly one command argument, got %d", len(rest))
			}

			env := append(os.Environ(), (*app).Sandbox.EnvFor(sb)...)
			argv := buildExecArgv(rest, useShell)

			return runExec(cmd.Context(), argv, env,
				os.Stdin, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	cmd.Flags().BoolVarP(&useShell, "shell", "s", false,
		"treat the single command arg as a shell string (sh -c)")
	return cmd
}

// buildExecArgv picks the argv to exec based on user input:
//
//   - no args → the user's shell (interactive subshell).
//   - --shell → /bin/sh -c <value>.
//   - otherwise → the args verbatim.
func buildExecArgv(args []string, useShell bool) []string {
	switch {
	case len(args) == 0:
		return []string{shellFromEnv()}
	case useShell:
		return []string{"/bin/sh", "-c", args[0]}
	default:
		return args
	}
}

// runExec launches argv with the given env and stdio, wiring
// [*exec.ExitError] through an ExecChildError so the child's exit code
// propagates to the parent process via ExitCode.
func runExec(ctx context.Context, argv, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
	//nolint:gosec // argv comes from the user's own shell invocation; we're the CLI, not a server.
	c := exec.CommandContext(ctx, argv[0], argv[1:]...)
	c.Env = env
	c.Stdin = stdin
	c.Stdout = stdout
	c.Stderr = stderr

	err := c.Run()
	if err == nil {
		return nil
	}
	var ex *exec.ExitError
	if errors.As(err, &ex) {
		return &ExecChildError{Code: ex.ExitCode()}
	}
	return fmt.Errorf("exec %s: %w", argv[0], err)
}

// shellFromEnv returns the user's preferred shell, falling back to
// /bin/sh when $SHELL is unset (rare: `env -i`, some container images).
func shellFromEnv() string {
	if s := os.Getenv("SHELL"); s != "" {
		return s
	}
	return "/bin/sh"
}
