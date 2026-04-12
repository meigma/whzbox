package cli

import (
	"context"
	"fmt"
	"os"
)

// Run executes the whzbox root command with the given context and
// returns the process exit code.
//
// It is the single shared entry point for production main() and any
// test harness that needs a main-equivalent (testscript, for example).
// Errors are printed to stderr once before the exit code is returned,
// matching the convention documented in DESIGN.md §12.
func Run(ctx context.Context) int {
	err := NewRootCommand().ExecuteContext(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}
	return ExitCode(err)
}
