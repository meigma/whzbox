package main

import (
	"context"
	"os"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"

	"github.com/meigma/whzbox/internal/cli"
)

// TestMain wires up testscript's subcommand runner. When a .txtar
// script invokes "whzbox ...", testscript calls the closure below
// instead of forking a real process — so tests exercise the exact
// binary code path without any compile step.
func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"whzbox": func() {
			os.Exit(cli.Run(context.Background()))
		},
	})
}

// TestScripts runs every .txtar file under testdata/script as an
// end-to-end test. Each script runs against a fresh temp directory
// (testscript's $WORK) and is free to mutate env vars like
// WHZBOX_STATE_DIR to isolate itself from the user's real state.
func TestScripts(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/script",
	})
}
