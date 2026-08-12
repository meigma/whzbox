package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/meigma/whzbox/internal/cli"
)

func TestRunWithOptions_WritesErrorsToInjectedStderr(t *testing.T) {
	var out, errOut bytes.Buffer
	options := cli.DefaultOptions()
	options.In = strings.NewReader("")
	options.Out = &out
	options.Err = &errOut
	options.Args = []string{"not-a-command"}

	code := cli.RunWithOptions(context.Background(), options)
	if code != cli.ExitGeneric {
		t.Errorf("exit code: got %d, want %d", code, cli.ExitGeneric)
	}
	if out.Len() != 0 {
		t.Errorf("stdout: got %q, want empty", out.String())
	}
	if !strings.Contains(errOut.String(), "Error:") {
		t.Errorf("stderr missing error: %q", errOut.String())
	}
}

func TestCommandsWithoutOperands_RejectArguments(t *testing.T) {
	for _, name := range []string{"login", "logout", "status", "list", "destroy", "version"} {
		t.Run(name, func(t *testing.T) {
			options := cli.DefaultOptions()
			options.Args = []string{name, "extra"}
			if err := cli.NewRootCommandWithOptions(options).Execute(); err == nil {
				t.Fatal("expected argument error")
			}
		})
	}
}
