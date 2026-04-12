package session

import (
	"errors"
	"time"
)

// Sentinel errors returned by the session service and its adapters.
// Callers should use [errors.Is] to test for these; underlying packages may
// wrap them with additional context.
var (
	// ErrInvalidCredentials is returned when the identity provider
	// rejects the supplied email/password pair.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrSessionExpired is returned when the stored refresh token is
	// too old to be exchanged for a new access token. Callers typically
	// react by re-prompting the user for their credentials.
	ErrSessionExpired = errors.New("session expired")

	// ErrPromptUnavailable is returned when the CLI needs to prompt the
	// user for credentials but no interactive terminal is attached.
	ErrPromptUnavailable = errors.New("no interactive terminal")

	// ErrUserAborted is returned when the user cancels an interactive
	// prompt (for example via ctrl-c).
	ErrUserAborted = errors.New("user aborted")
)

// Tokens is a snapshot of the Whizlabs access/refresh token pair for a
// user, annotated with absolute expiry times and the email it was issued
// to. It is the unit of persistence the TokenStore deals in.
type Tokens struct {
	AccessToken           string
	RefreshToken          string
	AccessTokenExpiresAt  time.Time
	RefreshTokenExpiresAt time.Time
	UserEmail             string
}

// AccessValid reports whether the access token is present and not yet
// expired at the given time. A token whose expiry equals now is treated
// as expired.
func (t Tokens) AccessValid(now time.Time) bool {
	return t.AccessToken != "" && now.Before(t.AccessTokenExpiresAt)
}

// AccessNearExpiry reports whether the access token either is already
// expired or will expire within the given window. An empty access token
// always counts as near expiry.
//
// This is the trigger the session service uses to proactively refresh
// before returning tokens to callers: we do not want a downstream API
// call to fail because the token lapsed while it was in flight.
func (t Tokens) AccessNearExpiry(now time.Time, window time.Duration) bool {
	if t.AccessToken == "" {
		return true
	}
	return !now.Before(t.AccessTokenExpiresAt.Add(-window))
}

// Refreshable reports whether a non-empty refresh token is present and
// still valid at the given time.
func (t Tokens) Refreshable(now time.Time) bool {
	return t.RefreshToken != "" && now.Before(t.RefreshTokenExpiresAt)
}
