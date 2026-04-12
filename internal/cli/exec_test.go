package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/meigma/whzbox/internal/core/sandbox"
)

// writeEnvDumper creates a tiny shell script in t.TempDir() that writes
// all AWS_* env vars to an output file and exits with the given code.
// Returns the script path and the output path, in that order.
func writeEnvDumper(t *testing.T, exitCode int) (string, string) {
	t.Helper()
	dir := t.TempDir()
	outfile := filepath.Join(dir, "env.out")
	script := filepath.Join(dir, "dump.sh")
	body := "#!/bin/sh\nenv | grep '^AWS_' > " + outfile + "\nexit " + strconv.Itoa(exitCode) + "\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return script, outfile
}

func TestExecCommand_NoCachedSandbox(t *testing.T) {
	now := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)
	app := newListTestApp(t, now)

	cmd := newExecCommand(&app)
	cmd.SetArgs([]string{"aws", "--", "/usr/bin/true"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no active aws sandbox") {
		t.Errorf("missing hint in %q", err.Error())
	}
	if !strings.Contains(err.Error(), "whzbox create aws") {
		t.Errorf("missing create hint in %q", err.Error())
	}
}

func TestExecCommand_ExpiredSandbox(t *testing.T) {
	now := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)
	expired := sampleListEntry("111111111111", now.Add(-time.Hour))
	app := newListTestApp(t, now, expired)

	cmd := newExecCommand(&app)
	cmd.SetArgs([]string{"aws", "--", "/usr/bin/true"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "no active aws sandbox") {
		t.Errorf("want no-active error, got %v", err)
	}
}

func TestExecCommand_UnknownKind(t *testing.T) {
	now := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)
	app := newListTestApp(t, now)

	cmd := newExecCommand(&app)
	cmd.SetArgs([]string{"gcp"})
	err := cmd.Execute()
	if !errors.Is(err, sandbox.ErrUnknownKind) {
		t.Errorf("want ErrUnknownKind, got %v", err)
	}
}

func TestExecCommand_DirectArgv_InjectsEnv(t *testing.T) {
	now := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)
	sb := sampleListEntry("111111111111", now.Add(time.Hour))
	app := newListTestApp(t, now, sb)
	script, outfile := writeEnvDumper(t, 0)

	cmd := newExecCommand(&app)
	cmd.SetArgs([]string{"aws", "--", script})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	env, err := os.ReadFile(outfile)
	if err != nil {
		t.Fatalf("read outfile: %v", err)
	}
	for _, want := range []string{
		"AWS_ACCESS_KEY_ID=AKIA_111111111111",
		"AWS_SECRET_ACCESS_KEY=sec_111111111111",
		"AWS_REGION=us-east-1",
		"AWS_DEFAULT_REGION=us-east-1",
	} {
		if !strings.Contains(string(env), want) {
			t.Errorf("missing %q in:\n%s", want, env)
		}
	}
}

func TestExecCommand_ChildNonZero_PropagatesExitCode(t *testing.T) {
	now := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)
	sb := sampleListEntry("111111111111", now.Add(time.Hour))
	app := newListTestApp(t, now, sb)
	script, _ := writeEnvDumper(t, 3)

	cmd := newExecCommand(&app)
	cmd.SetArgs([]string{"aws", "--", script})
	err := cmd.Execute()

	var child *ExecChildError
	if !errors.As(err, &child) {
		t.Fatalf("want *ExecChildError, got %T: %v", err, err)
	}
	if child.Code != 3 {
		t.Errorf("exit code: got %d, want 3", child.Code)
	}
	if got := ExitCode(err); got != 3 {
		t.Errorf("ExitCode: got %d, want 3", got)
	}
}

func TestExecCommand_Shell_RunsString(t *testing.T) {
	now := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)
	sb := sampleListEntry("111111111111", now.Add(time.Hour))
	app := newListTestApp(t, now, sb)

	cmd := newExecCommand(&app)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"aws", "-s", "printf '%s' \"$AWS_REGION\""})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got := out.String(); got != "us-east-1" {
		t.Errorf("shell output: got %q, want us-east-1", got)
	}
}

func TestExecCommand_Shell_RequiresSingleArg(t *testing.T) {
	now := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)
	sb := sampleListEntry("111111111111", now.Add(time.Hour))
	app := newListTestApp(t, now, sb)

	cmd := newExecCommand(&app)
	cmd.SetArgs([]string{"aws", "-s", "echo", "hi"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--shell takes exactly one") {
		t.Errorf("want usage error, got %v", err)
	}
}
