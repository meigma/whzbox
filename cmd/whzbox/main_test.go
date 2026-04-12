package main

import (
	"bytes"
	"testing"

	"github.com/meigma/whzbox/internal/cli"
)

func TestRootCommandHelp(t *testing.T) {
	cmd := cli.NewRootCommand()
	if cmd == nil {
		t.Fatal("NewRootCommand returned nil")
	}

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("--help returned error: %v", err)
	}

	if out.Len() == 0 {
		t.Error("expected --help to produce output")
	}
}
