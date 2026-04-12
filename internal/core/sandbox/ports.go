package sandbox

import (
	"context"
	"time"

	"github.com/meigma/whzbox/internal/core/session"
)

// Manager talks to the upstream sandbox broker (Whizlabs). The
// production implementation is the whizlabs HTTP adapter; tests use
// in-memory fakes.
//
// All methods take session.Tokens because the broker's API is scoped
// to the authenticated user, not a specific sandbox. The Manager is
// provider-agnostic — sandbox kind is identified by slug, which is
// passed through on Create and Commit.
type Manager interface {
	// Create provisions a new sandbox of the given slug and duration.
	// The returned Sandbox has credentials and console info filled in
	// but Identity is left empty — populating it is the Provider's job.
	Create(ctx context.Context, tokens session.Tokens, slug string, duration time.Duration) (*Sandbox, error)

	// Commit registers the newly-created sandbox with the user's
	// account. Without this call, Whizlabs does not track ownership
	// and Destroy/Active later return "Sandbox Not Found". This is
	// the "update" half of the two-phase provisioning dance we
	// discovered during feasibility testing.
	Commit(ctx context.Context, tokens session.Tokens, slug string, duration time.Duration) error

	// Destroy tears down the user's currently active sandbox.
	// Implementations must return ErrNoActiveSandbox (possibly wrapped)
	// when there is nothing to destroy, so callers can use errors.Is.
	Destroy(ctx context.Context, tokens session.Tokens) error

	// Active returns the user's currently-active sandbox, or
	// ErrNoActiveSandbox when nothing is active.
	Active(ctx context.Context, tokens session.Tokens) (*Sandbox, error)
}

// Provider knows the quirks of one sandbox kind: which Whizlabs slug
// to use when asking the Manager to create one, and how to verify the
// resulting credentials against the real cloud provider.
//
// Adding a new sandbox kind means implementing one Provider (under
// internal/adapters/<kind>verify/) and registering it in the CLI's
// App container.
type Provider interface {
	Kind() Kind
	Slug() string
	VerifyCredentials(ctx context.Context, creds Credentials) (Identity, error)
}

// Store caches provisioned sandboxes on disk so that a subsequent
// `create <kind>` can reuse an already-live sandbox instead of asking
// Whizlabs for a fresh one. The Whizlabs API has no endpoint for
// refetching credentials of an active sandbox, so without this cache
// the creds printed by `create` are lost the moment the user closes
// the terminal.
//
// Load returns (nil, false, nil) when nothing is cached for the kind.
// ClearAll removes every cached sandbox — there is only one active
// sandbox per Whizlabs account, so a successful Destroy clears all
// kinds regardless of which one was destroyed.
type Store interface {
	Load(ctx context.Context, kind Kind) (*Sandbox, bool, error)
	Save(ctx context.Context, sb *Sandbox) error
	ClearAll(ctx context.Context) error
}

// SessionAuthorizer is the narrow slice of session.Service that the
// sandbox service depends on. Declaring it here (rather than depending
// on *session.Service directly) keeps sandbox unit tests free of fake
// provider/store/prompt machinery — they supply a two-line stub
// instead.
//
// session.Service satisfies this interface by construction; no
// adapter code is needed.
type SessionAuthorizer interface {
	EnsureValid(ctx context.Context) (session.Tokens, error)
}
