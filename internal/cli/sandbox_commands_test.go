package cli

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/meigma/whzbox/internal/core/clock"
	"github.com/meigma/whzbox/internal/core/sandbox"
	"github.com/meigma/whzbox/internal/core/session"
)

// -----------------------------------------------------------------------------
// Stub sandbox ports
// -----------------------------------------------------------------------------

type stubManager struct {
	createResult *sandbox.Sandbox
	createErr    error

	commitErr error

	destroyErr error

	activeResult *sandbox.Sandbox
	activeErr    error
}

func (s *stubManager) Create(
	_ context.Context,
	_ session.Tokens,
	slug string,
	_ time.Duration,
) (*sandbox.Sandbox, error) {
	if s.createResult != nil {
		s.createResult.Slug = slug
	}
	return s.createResult, s.createErr
}

func (s *stubManager) Commit(_ context.Context, _ session.Tokens, _ string, _ time.Duration) error {
	return s.commitErr
}

func (s *stubManager) Destroy(_ context.Context, _ session.Tokens) error {
	return s.destroyErr
}

func (s *stubManager) Active(_ context.Context, _ session.Tokens) (*sandbox.Sandbox, error) {
	return s.activeResult, s.activeErr
}

type stubVerifier struct {
	identity sandbox.Identity
	err      error
}

func (s *stubVerifier) Kind() sandbox.Kind { return sandbox.KindAWS }
func (s *stubVerifier) Slug() string       { return "aws-sandbox" }
func (s *stubVerifier) VerifyCredentials(_ context.Context, _ sandbox.Credentials) (sandbox.Identity, error) {
	return s.identity, s.err
}

// stubAuth is a SessionAuthorizer that always returns a valid Tokens.
type stubAuth struct {
	err error
}

func (s *stubAuth) EnsureValid(_ context.Context) (session.Tokens, error) {
	if s.err != nil {
		return session.Tokens{}, s.err
	}
	return session.Tokens{AccessToken: "tok", UserEmail: "alice@example.com"}, nil
}

func newSandboxTestApp(mgr *stubManager, ver *stubVerifier, opts ...func(*App)) *App {
	svc := sandbox.NewService(
		&stubAuth{},
		mgr,
		map[sandbox.Kind]sandbox.Provider{sandbox.KindAWS: ver},
		clock.Real{},
		nil,
	)
	app := &App{Sandbox: svc}
	for _, o := range opts {
		o(app)
	}
	return app
}

// -----------------------------------------------------------------------------
// create command
// -----------------------------------------------------------------------------

func sampleCreated() *sandbox.Sandbox {
	return &sandbox.Sandbox{
		Credentials: sandbox.Credentials{
			AccessKey: "AKIA_TEST",
			SecretKey: "sec",
		},
		Console: sandbox.Console{
			URL:      "https://111.signin.aws.amazon.com/console",
			Username: "whiz_user",
			Password: "pw",
		},
		StartedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
}

func TestCreateCommand_Success(t *testing.T) {
	mgr := &stubManager{createResult: sampleCreated()}
	ver := &stubVerifier{identity: sandbox.Identity{
		Account: "111111111111",
		UserID:  "AIDAT",
		ARN:     "arn:aws:iam::111111111111:user/whiz_user",
		Region:  "us-east-1",
	}}
	app := newSandboxTestApp(mgr, ver)

	cmd := newCreateCommand(&app)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"aws"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	got := out.String()
	for _, want := range []string{"AKIA_TEST", "sec", "arn:aws:iam::111111111111", "whzbox destroy"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestCreateCommand_VerificationFailure_StillRendersBox(t *testing.T) {
	mgr := &stubManager{createResult: sampleCreated()}
	ver := &stubVerifier{err: errors.New("InvalidClientTokenId")}
	app := newSandboxTestApp(mgr, ver, func(a *App) {
		// create.go calls (*app).Logger.Warn on verification
		// failure, so the Logger field must be non-nil.
		a.Logger = slog.New(slog.DiscardHandler)
	})

	cmd := newCreateCommand(&app)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"aws"})
	err := cmd.Execute()
	if !errors.Is(err, sandbox.ErrVerificationFailed) {
		t.Errorf("error: got %v, want ErrVerificationFailed", err)
	}
	if got := ExitCode(err); got != ExitProvider {
		t.Errorf("ExitCode: got %d, want %d", got, ExitProvider)
	}
	// The credential box must still be rendered.
	if !strings.Contains(out.String(), "AKIA_TEST") {
		t.Errorf("verification failure should still render box:\n%s", out.String())
	}
}

func TestCreateCommand_InvalidProvider(t *testing.T) {
	mgr := &stubManager{}
	ver := &stubVerifier{}
	app := newSandboxTestApp(mgr, ver)

	cmd := newCreateCommand(&app)
	cmd.SetArgs([]string{"gcp"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid provider")
	}
}

func TestCreateCommand_ManagerCreateError(t *testing.T) {
	bang := errors.New("whizlabs 500")
	mgr := &stubManager{createErr: bang}
	ver := &stubVerifier{}
	app := newSandboxTestApp(mgr, ver)

	cmd := newCreateCommand(&app)
	cmd.SetArgs([]string{"aws"})
	err := cmd.Execute()
	if !errors.Is(err, bang) {
		t.Errorf("error: got %v, want wrapped %v", err, bang)
	}
	if got := ExitCode(err); got != ExitProvider {
		t.Errorf("ExitCode: got %d, want %d", got, ExitProvider)
	}
}

// -----------------------------------------------------------------------------
// destroy command
// -----------------------------------------------------------------------------

func TestDestroyCommand_Yes_Success(t *testing.T) {
	mgr := &stubManager{}
	ver := &stubVerifier{}
	app := newSandboxTestApp(mgr, ver, func(a *App) {
		a.Config.AssumeYes = true
	})

	cmd := newDestroyCommand(&app)
	if err := cmd.Execute(); err != nil {
		t.Errorf("Execute: %v", err)
	}
}

func TestDestroyCommand_Yes_NoActiveSandbox(t *testing.T) {
	mgr := &stubManager{destroyErr: sandbox.ErrNoActiveSandbox}
	ver := &stubVerifier{}
	app := newSandboxTestApp(mgr, ver, func(a *App) {
		a.Config.AssumeYes = true
	})

	cmd := newDestroyCommand(&app)
	err := cmd.Execute()
	if !errors.Is(err, sandbox.ErrNoActiveSandbox) {
		t.Errorf("error: got %v, want ErrNoActiveSandbox", err)
	}
}

func TestDestroyCommand_Yes_ManagerError(t *testing.T) {
	bang := errors.New("whizlabs 500")
	mgr := &stubManager{destroyErr: bang}
	ver := &stubVerifier{}
	app := newSandboxTestApp(mgr, ver, func(a *App) {
		a.Config.AssumeYes = true
	})

	cmd := newDestroyCommand(&app)
	err := cmd.Execute()
	if !errors.Is(err, bang) {
		t.Errorf("error: got %v, want wrapped %v", err, bang)
	}
	if got := ExitCode(err); got != ExitProvider {
		t.Errorf("ExitCode: got %d, want %d", got, ExitProvider)
	}
}

func TestDestroyCommand_NoTTY_RequiresYes(t *testing.T) {
	mgr := &stubManager{}
	ver := &stubVerifier{}
	app := newSandboxTestApp(mgr, ver) // AssumeYes=false

	cmd := newDestroyCommand(&app)
	// go test runs with stdin/stderr not attached to a terminal, so
	// the interactive confirm short-circuits with ErrPromptUnavailable.
	err := cmd.Execute()
	if !errors.Is(err, session.ErrPromptUnavailable) {
		t.Errorf("error: got %v, want ErrPromptUnavailable", err)
	}
}

// -----------------------------------------------------------------------------
// status command
// -----------------------------------------------------------------------------

func TestStatusCommand_NotLoggedIn(t *testing.T) {
	// Status reads via Session.Current which hits the store. Build a
	// real session.Service with an empty in-memory store.
	store := &memStore{}
	svc := session.NewService(&noopProvider{}, store, &noopPrompt{}, clock.Real{}, nil)
	app := &App{Session: svc}

	cmd := newStatusCommand(&app)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out.String(), "(not logged in)") {
		t.Errorf("output missing not-logged-in marker:\n%s", out.String())
	}
	if strings.Contains(out.String(), "Active sandbox") {
		t.Errorf("status should be session-only:\n%s", out.String())
	}
}

func TestStatusCommand_LoggedIn(t *testing.T) {
	store := &memStore{
		tokens: session.Tokens{
			UserEmail:            "alice@example.com",
			AccessToken:          "tok",
			AccessTokenExpiresAt: time.Now().Add(12 * time.Hour),
		},
		found: true,
	}
	svc := session.NewService(&noopProvider{}, store, &noopPrompt{}, clock.Real{}, nil)
	app := &App{Session: svc}

	cmd := newStatusCommand(&app)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !strings.Contains(out.String(), "alice@example.com") {
		t.Errorf("output missing email:\n%s", out.String())
	}
	if strings.Contains(out.String(), "Active sandbox") {
		t.Errorf("status should be session-only:\n%s", out.String())
	}
}

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

type memStore struct {
	tokens session.Tokens
	found  bool
}

func (m *memStore) Load(_ context.Context) (session.Tokens, bool, error) {
	return m.tokens, m.found, nil
}
func (m *memStore) Save(_ context.Context, _ session.Tokens) error { return nil }
func (m *memStore) Clear(_ context.Context) error                  { return nil }

type noopProvider struct{}

func (noopProvider) Login(_ context.Context, _, _ string) (session.Tokens, error) {
	return session.Tokens{}, errors.New("noop")
}
func (noopProvider) Refresh(_ context.Context, _ session.Tokens) (session.Tokens, error) {
	return session.Tokens{}, errors.New("noop")
}

type noopPrompt struct{}

func (noopPrompt) Credentials(_ context.Context, _ string) (string, string, error) {
	return "", "", errors.New("noop")
}
