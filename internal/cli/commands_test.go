package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/meigma/whzbox/internal/core/clock"
	"github.com/meigma/whzbox/internal/core/session"
)

// Minimal port stubs to build a real session.Service under test without
// touching real adapters. Each stub accepts hand-tuned error fields that
// drive the specific scenario the test cares about.

type stubProvider struct {
	loginTokens session.Tokens
	loginErr    error
}

func (s *stubProvider) Login(_ context.Context, _, _ string) (session.Tokens, error) {
	return s.loginTokens, s.loginErr
}

func (s *stubProvider) Refresh(_ context.Context, _ session.Tokens) (session.Tokens, error) {
	return session.Tokens{}, errors.New("refresh not used in these tests")
}

type stubStore struct {
	saved session.Tokens
	saves int
}

func (s *stubStore) Load(_ context.Context) (session.Tokens, bool, error) {
	return session.Tokens{}, false, nil
}

func (s *stubStore) Save(_ context.Context, t session.Tokens) error {
	s.saves++
	s.saved = t
	return nil
}

func (s *stubStore) Clear(_ context.Context) error { return nil }

type stubPrompt struct {
	email    string
	password string
	err      error
}

func (s *stubPrompt) Credentials(_ context.Context, _ string) (string, string, error) {
	if s.err != nil {
		return "", "", s.err
	}
	return s.email, s.password, nil
}

func newTestApp(provider *stubProvider, prompt *stubPrompt) *App {
	store := &stubStore{}
	svc := session.NewService(provider, store, prompt, clock.Real{}, nil)
	return &App{Session: svc}
}

// -----------------------------------------------------------------------------
// login command
// -----------------------------------------------------------------------------

func TestLoginCommand_Success(t *testing.T) {
	app := newTestApp(
		&stubProvider{loginTokens: session.Tokens{UserEmail: "alice@example.com"}},
		&stubPrompt{email: "alice@example.com", password: "pw"},
	)
	cmd := newLoginCommand(&app)
	if err := cmd.Execute(); err != nil {
		t.Errorf("Execute: %v", err)
	}
}

func TestLoginCommand_InvalidCredentials(t *testing.T) {
	app := newTestApp(
		&stubProvider{loginErr: session.ErrInvalidCredentials},
		&stubPrompt{email: "a", password: "b"},
	)
	cmd := newLoginCommand(&app)
	err := cmd.Execute()
	if !errors.Is(err, session.ErrInvalidCredentials) {
		t.Errorf("error: got %v, want ErrInvalidCredentials", err)
	}
	if got := ExitCode(err); got != ExitAuth {
		t.Errorf("ExitCode: got %d, want %d", got, ExitAuth)
	}
}

func TestLoginCommand_UserAborted(t *testing.T) {
	app := newTestApp(
		&stubProvider{},
		&stubPrompt{err: session.ErrUserAborted},
	)
	cmd := newLoginCommand(&app)
	err := cmd.Execute()
	if !errors.Is(err, session.ErrUserAborted) {
		t.Errorf("error: got %v, want ErrUserAborted", err)
	}
	if got := ExitCode(err); got != ExitUserAborted {
		t.Errorf("ExitCode: got %d, want %d", got, ExitUserAborted)
	}
}

func TestLoginCommand_PromptUnavailable(t *testing.T) {
	app := newTestApp(
		&stubProvider{},
		&stubPrompt{err: session.ErrPromptUnavailable},
	)
	cmd := newLoginCommand(&app)
	err := cmd.Execute()
	if !errors.Is(err, session.ErrPromptUnavailable) {
		t.Errorf("error: got %v, want ErrPromptUnavailable", err)
	}
	if got := ExitCode(err); got != ExitNonInteractive {
		t.Errorf("ExitCode: got %d, want %d", got, ExitNonInteractive)
	}
}

// -----------------------------------------------------------------------------
// logout command
// -----------------------------------------------------------------------------

func TestLogoutCommand_Success(t *testing.T) {
	app := newTestApp(&stubProvider{}, &stubPrompt{})
	cmd := newLogoutCommand(&app)
	if err := cmd.Execute(); err != nil {
		t.Errorf("Execute: %v", err)
	}
}
