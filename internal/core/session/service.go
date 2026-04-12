package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/meigma/whzbox/internal/core/clock"
)

// AccessTokenRefreshWindow is how far in advance of access-token expiry
// the service will proactively refresh. Chosen to comfortably outlast
// the slowest downstream API call a user might make in a single command.
const AccessTokenRefreshWindow = 10 * time.Minute

// Service coordinates the identity provider, token store, and credential
// prompt to hand out valid tokens to the rest of the CLI.
//
// It has no I/O of its own; every external interaction goes through one
// of the injected ports. This is what makes the session domain unit-
// testable in isolation from HTTP, the filesystem, or a terminal.
type Service struct {
	provider IdentityProvider
	store    TokenStore
	prompt   Prompt
	clock    clock.Clock
	logger   *slog.Logger
}

// NewService returns a Service wired up with the supplied ports. All
// port arguments are required; passing nil will panic at first use. A
// nil logger is replaced with a discard handler so callers that do not
// care about logs can ignore the argument.
func NewService(
	provider IdentityProvider,
	store TokenStore,
	prompt Prompt,
	c clock.Clock,
	logger *slog.Logger,
) *Service {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Service{
		provider: provider,
		store:    store,
		prompt:   prompt,
		clock:    c,
		logger:   logger,
	}
}

// EnsureValid returns a Tokens value that is safe to use right now.
//
// The resolution order is:
//
//  1. Cache hit, access token not within the refresh window -> return cached.
//  2. Cache hit, within refresh window, refresh token still valid -> refresh.
//  3. Cache hit, refresh failed or not refreshable -> prompt and re-login.
//  4. Cache miss -> prompt and login.
//
// When the prompt is unavailable (non-interactive stdin), ErrPromptUnavailable
// is returned to the caller. When the user aborts the prompt, ErrUserAborted
// is returned. Both errors are sentinel-wrapped so callers can use [errors.Is].
func (s *Service) EnsureValid(ctx context.Context) (Tokens, error) {
	now := s.clock.Now()

	stored, found, err := s.store.Load(ctx)
	if err != nil {
		return Tokens{}, fmt.Errorf("load session: %w", err)
	}

	if found && stored.AccessValid(now) && !stored.AccessNearExpiry(now, AccessTokenRefreshWindow) {
		s.logger.DebugContext(ctx, "session loaded from cache",
			"user", stored.UserEmail,
			"expires", stored.AccessTokenExpiresAt,
		)
		return stored, nil
	}

	if found && stored.Refreshable(now) {
		s.logger.DebugContext(ctx, "refreshing session", "user", stored.UserEmail)
		refreshed, rerr := s.provider.Refresh(ctx, stored)
		if rerr == nil {
			if serr := s.store.Save(ctx, refreshed); serr != nil {
				return Tokens{}, fmt.Errorf("save refreshed session: %w", serr)
			}
			s.logger.InfoContext(ctx, "session refreshed",
				"user", refreshed.UserEmail,
				"expires", refreshed.AccessTokenExpiresAt,
			)
			return refreshed, nil
		}
		s.logger.WarnContext(ctx, "session refresh failed", "err", rerr)
	}

	return s.loginViaPrompt(ctx, stored.UserEmail)
}

// Login always prompts the user for credentials, calls the identity
// provider, and replaces whatever is in the store. It is the
// implementation of `whzbox login` — an explicit "start fresh" command
// that ignores any cached refresh token.
func (s *Service) Login(ctx context.Context) (Tokens, error) {
	stored, _, _ := s.store.Load(ctx) // best-effort default-email hint
	return s.loginViaPrompt(ctx, stored.UserEmail)
}

// Logout removes any stored tokens. Clearing an empty store is not an
// error — it is the idempotent "make sure I am logged out" semantic.
func (s *Service) Logout(ctx context.Context) error {
	if err := s.store.Clear(ctx); err != nil {
		return fmt.Errorf("clear session: %w", err)
	}
	s.logger.InfoContext(ctx, "session cleared")
	return nil
}

// Current returns the currently stored tokens without refreshing or
// prompting. It is the read-only variant used by `whzbox status` so
// running `status` never triggers an auth flow.
//
// The second return value is false when nothing is stored — the
// normal "never logged in" state.
func (s *Service) Current(ctx context.Context) (Tokens, bool, error) {
	return s.store.Load(ctx)
}

// loginViaPrompt is the shared tail of EnsureValid and Login: run the
// credential prompt, hit the identity provider, persist the result.
func (s *Service) loginViaPrompt(ctx context.Context, defaultEmail string) (Tokens, error) {
	email, password, err := s.prompt.Credentials(ctx, defaultEmail)
	if err != nil {
		// Sentinel errors pass through unwrapped so callers can
		// errors.Is them directly without walking the chain twice.
		if errors.Is(err, ErrPromptUnavailable) || errors.Is(err, ErrUserAborted) {
			return Tokens{}, err
		}
		return Tokens{}, fmt.Errorf("credentials prompt: %w", err)
	}

	tokens, err := s.provider.Login(ctx, email, password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return Tokens{}, err
		}
		return Tokens{}, fmt.Errorf("login: %w", err)
	}

	err = s.store.Save(ctx, tokens)
	if err != nil {
		return Tokens{}, fmt.Errorf("save session: %w", err)
	}
	s.logger.InfoContext(ctx, "session established",
		"user", tokens.UserEmail,
		"expires", tokens.AccessTokenExpiresAt,
	)
	return tokens, nil
}
