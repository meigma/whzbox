// Command whzbox spins up on-demand cloud sandboxes through Whizlabs,
// fetches their credentials, and verifies they work.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/meigma/whzbox/internal/cli"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return cli.Run(ctx)
}
