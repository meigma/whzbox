package session

import "context"

// IdentityProvider exchanges credentials for access/refresh tokens and
// refreshes existing tokens. The production implementation is the
// whizlabs HTTP adapter; tests use in-memory fakes.
type IdentityProvider interface {
	// Login exchanges an email/password pair for a fresh Tokens value.
	// Implementations must return ErrInvalidCredentials (possibly
	// wrapped) when the upstream rejects the credentials, so callers
	// can react without inspecting error strings.
	Login(ctx context.Context, email, password string) (Tokens, error)

	// Refresh exchanges a currently-valid or recently-expired token pair
	// for a fresh one. Implementations must return ErrSessionExpired
	// (possibly wrapped) when the refresh token is too old to be
	// accepted.
	Refresh(ctx context.Context, current Tokens) (Tokens, error)
}

// TokenStore persists Tokens across CLI invocations. The production
// implementation is the xdgstore file-backed adapter; tests use in-memory
// fakes.
type TokenStore interface {
	// Load returns the stored tokens. The second return value is false
	// when nothing has been stored yet — this is NOT an error, it is the
	// normal "first run" state. Errors are reserved for I/O failures
	// and corrupt state.
	Load(ctx context.Context) (Tokens, bool, error)

	// Save writes the supplied tokens, replacing any prior state.
	Save(ctx context.Context, t Tokens) error

	// Clear removes any stored tokens. Clearing an empty store is a
	// no-op and must not return an error.
	Clear(ctx context.Context) error
}

// Prompt asks the user for credentials via an interactive TUI. The
// production implementation is huhprompt; tests use in-memory fakes.
type Prompt interface {
	// Credentials asks the user for an email and password. The
	// defaultEmail hint is prefilled when non-empty; implementations
	// should treat it as a prefilled field the user can overwrite.
	//
	// Implementations must return ErrPromptUnavailable when no TTY is
	// attached, and ErrUserAborted when the user dismisses the prompt.
	Credentials(ctx context.Context, defaultEmail string) (email, password string, err error)
}
