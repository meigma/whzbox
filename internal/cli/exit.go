package cli

import (
	"errors"
	"fmt"

	"github.com/meigma/whzbox/internal/core/sandbox"
	"github.com/meigma/whzbox/internal/core/session"
)

// Exit codes, documented in DESIGN.md §12. Scripts can branch on these.
const (
	ExitOK             = 0
	ExitGeneric        = 1
	ExitAuth           = 2
	ExitProvider       = 3
	ExitUserAborted    = 4
	ExitNonInteractive = 5
)

// ExecChildError wraps a non-zero exit code from a child process
// launched by `whzbox exec`. Run() treats this specially: the child's
// exit code propagates to the parent process, and no "Error:" line is
// printed because the child has already written whatever it wanted to
// its stdio.
type ExecChildError struct {
	Code int
}

func (e *ExecChildError) Error() string {
	return fmt.Sprintf("child process exited with code %d", e.Code)
}

// ExitCode maps an error returned from command execution to a process
// exit code. main.go calls this as its last step before [os.Exit].
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	var execChild *ExecChildError
	if errors.As(err, &execChild) {
		return execChild.Code
	}
	switch {
	case errors.Is(err, session.ErrInvalidCredentials),
		errors.Is(err, session.ErrSessionExpired):
		return ExitAuth
	case errors.Is(err, sandbox.ErrProvider),
		errors.Is(err, sandbox.ErrVerificationFailed),
		errors.Is(err, sandbox.ErrNoActiveSandbox):
		return ExitProvider
	case errors.Is(err, session.ErrUserAborted):
		return ExitUserAborted
	case errors.Is(err, session.ErrPromptUnavailable):
		return ExitNonInteractive
	default:
		return ExitGeneric
	}
}
