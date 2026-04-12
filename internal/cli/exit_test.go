package cli_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/meigma/whzbox/internal/cli"
	"github.com/meigma/whzbox/internal/core/sandbox"
	"github.com/meigma/whzbox/internal/core/session"
)

func TestExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, cli.ExitOK},
		{"invalid credentials", session.ErrInvalidCredentials, cli.ExitAuth},
		{"session expired", session.ErrSessionExpired, cli.ExitAuth},
		{"provider", sandbox.ErrProvider, cli.ExitProvider},
		{"verification failed", sandbox.ErrVerificationFailed, cli.ExitProvider},
		{"no active sandbox", sandbox.ErrNoActiveSandbox, cli.ExitProvider},
		{"user aborted", session.ErrUserAborted, cli.ExitUserAborted},
		{"prompt unavailable", session.ErrPromptUnavailable, cli.ExitNonInteractive},
		{"generic", errors.New("boom"), cli.ExitGeneric},
		// Wrapped errors should still match their sentinel.
		{"wrapped invalid credentials", fmt.Errorf("login failed: %w", session.ErrInvalidCredentials), cli.ExitAuth},
		{"wrapped provider", fmt.Errorf("create failed: %w", sandbox.ErrProvider), cli.ExitProvider},
		{"wrapped prompt unavailable", fmt.Errorf("shell: %w", session.ErrPromptUnavailable), cli.ExitNonInteractive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cli.ExitCode(tt.err)
			if got != tt.want {
				t.Errorf("ExitCode(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}
