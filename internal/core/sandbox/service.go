package sandbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/meigma/whzbox/internal/core/clock"
)

// Service is the sandbox use case layer. It coordinates the session
// authorizer (for auth), the Manager (for talking to Whizlabs), and
// per-kind Providers (for verifying credentials).
//
// It has no I/O of its own; every external interaction goes through
// one of the injected ports, which makes the create/destroy/status
// flows unit-testable in isolation.
type Service struct {
	session   SessionAuthorizer
	manager   Manager
	providers map[Kind]Provider
	store     Store
	clock     clock.Clock
	logger    *slog.Logger
}

// NewService returns a Service wired up with the supplied ports. All
// port arguments are required; passing nil will panic at first use.
// A nil logger is replaced with a discard handler so callers that do
// not care about logs can ignore the argument.
func NewService(
	auth SessionAuthorizer,
	manager Manager,
	providers map[Kind]Provider,
	store Store,
	c clock.Clock,
	logger *slog.Logger,
) *Service {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Service{
		session:   auth,
		manager:   manager,
		providers: providers,
		store:     store,
		clock:     c,
		logger:    logger,
	}
}

// Create provisions a sandbox of the given kind, commits it to the
// user's account, and verifies the returned credentials against the
// underlying cloud provider.
//
// The flow is:
//
//  1. Look up the Provider for the kind (fail fast on bad input).
//  2. Ensure the session is valid (may refresh or re-prompt).
//  3. Call Manager.Create to provision and get credentials.
//  4. Call Manager.Commit to register the sandbox with the user.
//     If this fails, best-effort Destroy and return the original
//     commit error wrapped.
//  5. Call Provider.VerifyCredentials to confirm the credentials work.
//     If this fails, return the created sandbox alongside
//     ErrVerificationFailed wrapping the underlying error — the
//     sandbox is real and already billed against the account, so
//     surface it to the user rather than silently destroying it.
//
// The returned Sandbox always has Kind and Slug populated when non-nil.
func (s *Service) Create(ctx context.Context, kind Kind, duration time.Duration) (*Sandbox, error) {
	prov, ok := s.providers[kind]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownKind, kind)
	}

	if cached, hit, lerr := s.store.Load(ctx, kind); lerr != nil {
		s.logger.WarnContext(ctx, "sandbox cache load failed; continuing without reuse", "err", lerr)
	} else if hit && cached.ExpiresAt.After(s.clock.Now()) {
		s.logger.InfoContext(ctx, "reusing active sandbox",
			"kind", cached.Kind, "expires_at", cached.ExpiresAt)
		return cached, nil
	}

	tokens, err := s.session.EnsureValid(ctx)
	if err != nil {
		return nil, err
	}

	sb, err := s.manager.Create(ctx, tokens, prov.Slug(), duration)
	if err != nil {
		return nil, fmt.Errorf("%w: create sandbox: %w", ErrProvider, err)
	}
	sb.Kind = kind
	sb.Slug = prov.Slug()

	if cerr := s.manager.Commit(ctx, tokens, prov.Slug(), duration); cerr != nil {
		// The sandbox was provisioned on the cloud side but the
		// broker didn't register ownership. Try to tear it down so
		// we don't leak a phantom sandbox. If that also fails, log
		// the secondary error and still return the original cause.
		if derr := s.manager.Destroy(ctx, tokens); derr != nil {
			s.logger.ErrorContext(ctx, "commit failed and cleanup destroy also failed",
				"commit_err", cerr,
				"destroy_err", derr,
			)
		}
		return nil, fmt.Errorf("%w: commit sandbox: %w", ErrProvider, cerr)
	}

	identity, verr := prov.VerifyCredentials(ctx, sb.Credentials)
	if verr == nil {
		sb.Identity = identity
	}

	// Cache the provisioned sandbox so a subsequent create can reuse
	// it. The upstream creds are real and billed against the account
	// even when verification fails, so we cache either way. A cache
	// write failure must not mask the successful provision.
	if cerr := s.store.Save(ctx, sb); cerr != nil {
		s.logger.WarnContext(ctx, "failed to cache sandbox", "err", cerr)
	}

	if verr != nil {
		// Return the sandbox so the CLI can still render the
		// credentials — verification failure is a warning, not a
		// destructive event. The error is sentinel-wrapped so the
		// caller can errors.Is it.
		return sb, fmt.Errorf("%w: %w", ErrVerificationFailed, verr)
	}
	return sb, nil
}

// Destroy tears down the user's currently active sandbox. It returns
// ErrNoActiveSandbox (possibly wrapped by the Manager) when there is
// nothing to destroy; callers can use [errors.Is] to distinguish this
// from other failures.
func (s *Service) Destroy(ctx context.Context) error {
	tokens, err := s.session.EnsureValid(ctx)
	if err != nil {
		return err
	}
	err = s.manager.Destroy(ctx, tokens)
	if err != nil {
		if errors.Is(err, ErrNoActiveSandbox) {
			return err
		}
		return fmt.Errorf("%w: destroy sandbox: %w", ErrProvider, err)
	}
	if cerr := s.store.ClearAll(ctx); cerr != nil {
		s.logger.WarnContext(ctx, "failed to clear sandbox cache", "err", cerr)
	}
	s.logger.InfoContext(ctx, "sandbox destroyed")
	return nil
}

// Status returns the user's currently active sandbox, or (nil, nil)
// when there is nothing active.
//
// Status does not re-verify credentials — verification is a
// heavyweight operation (IAM propagation retries, STS round-trips)
// that only makes sense on Create. A stale Status call would be
// misleadingly slow.
func (s *Service) Status(ctx context.Context) (*Sandbox, error) {
	tokens, err := s.session.EnsureValid(ctx)
	if err != nil {
		return nil, err
	}
	sb, err := s.manager.Active(ctx, tokens)
	if err != nil {
		if errors.Is(err, ErrNoActiveSandbox) {
			return nil, nil //nolint:nilnil // deliberate API: nil,nil means "no active sandbox"
		}
		return nil, fmt.Errorf("%w: status sandbox: %w", ErrProvider, err)
	}
	return sb, nil
}
