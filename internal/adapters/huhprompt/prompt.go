package huhprompt

import (
	"context"
	"errors"
	"fmt"
	"os"

	"charm.land/huh/v2"

	"github.com/meigma/whzbox/internal/core/session"
	"github.com/meigma/whzbox/internal/ui"
)

// Prompt implements session.Prompt using charm.land/huh/v2 forms on
// [os.Stdin] and [os.Stderr].
//
// The huh form is blocking: Credentials returns once the user submits,
// cancels, or the context is cancelled. Forms inherit colour and theming
// from ui.HuhTheme so the look matches the rest of the CLI.
type Prompt struct{}

// New returns a ready-to-use Prompt. It takes no arguments because all
// relevant state lives inside the huh library.
func New() *Prompt {
	return &Prompt{}
}

// Credentials implements session.Prompt. It first checks that both
// stdin and stderr are attached to a terminal; if either is not, it
// returns session.ErrPromptUnavailable so callers can fail fast in
// non-interactive environments (CI, pipes) instead of hanging.
//
// When the user cancels the form (e.g. ctrl-c), session.ErrUserAborted
// is returned. Other errors are wrapped with context.
func (*Prompt) Credentials(ctx context.Context, defaultEmail string) (string, string, error) {
	if !ui.IsInteractive(os.Stdin) || !ui.IsInteractive(os.Stderr) {
		return "", "", session.ErrPromptUnavailable
	}

	email := defaultEmail
	var password string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Email").
				Value(&email).
				Validate(requireNonEmpty("email")),
			huh.NewInput().
				Title("Password").
				EchoMode(huh.EchoModePassword).
				Value(&password).
				Validate(requireNonEmpty("password")),
		),
	).WithTheme(huh.ThemeFunc(ui.HuhTheme))

	if err := form.RunWithContext(ctx); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return "", "", session.ErrUserAborted
		}
		return "", "", fmt.Errorf("credentials form: %w", err)
	}

	return email, password, nil
}

func requireNonEmpty(name string) func(string) error {
	return func(s string) error {
		if s == "" {
			return fmt.Errorf("%s is required", name)
		}
		return nil
	}
}
