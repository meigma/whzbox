package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/meigma/whzbox/internal/cli"
)

func TestVersionCommand_PrintsBuildInfo(t *testing.T) {
	var out bytes.Buffer
	options := cli.DefaultOptions()
	options.Out = &out
	options.Err = &out
	options.Build = cli.BuildInfo{
		Version: "v1.2.3",
		Commit:  "abc123",
		Time:    "2026-04-11T00:00:00Z",
	}
	cmd := cli.NewRootCommandWithOptions(options)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("version returned error: %v", err)
	}

	got := out.String()
	for _, want := range []string{"whzbox", "v1.2.3", "abc123", "2026-04-11T00:00:00Z"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q: %s", want, got)
		}
	}
}

func TestVersionCommand_DoesNotRequireStateDir(t *testing.T) {
	t.Setenv("WHZBOX_STATE_DIR", "/dev/null")

	cmd := cli.NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("version returned error: %v", err)
	}
	if !strings.Contains(out.String(), "whzbox") {
		t.Errorf("output missing version line: %s", out.String())
	}
}

func TestRootCommand_VersionFlag(t *testing.T) {
	options := cli.DefaultOptions()
	options.Build = cli.BuildInfo{Version: "v9.9.9", Commit: "abc", Time: "today"}
	cmd := cli.NewRootCommandWithOptions(options)
	if !strings.Contains(cmd.Version, "v9.9.9") {
		t.Errorf("version field: got %q", cmd.Version)
	}
}
