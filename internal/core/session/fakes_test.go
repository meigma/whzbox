package session_test

import (
	"context"
	"time"

	"github.com/meigma/whzbox/internal/core/session"
)

// fakeProvider is an in-memory session.IdentityProvider for unit tests.
// Fields prefixed "login*" / "refresh*" control return values; call
// counters and the most recent arguments are exposed for assertions.
type fakeProvider struct {
	loginResult session.Tokens
	loginErr    error

	refreshResult session.Tokens
	refreshErr    error

	loginCalls        int
	refreshCalls      int
	lastEmail         string
	lastPassword      string
	lastRefreshedFrom session.Tokens
}

func (f *fakeProvider) Login(_ context.Context, email, password string) (session.Tokens, error) {
	f.loginCalls++
	f.lastEmail = email
	f.lastPassword = password
	return f.loginResult, f.loginErr
}

func (f *fakeProvider) Refresh(_ context.Context, current session.Tokens) (session.Tokens, error) {
	f.refreshCalls++
	f.lastRefreshedFrom = current
	return f.refreshResult, f.refreshErr
}

// fakeStore is an in-memory session.TokenStore for unit tests.
type fakeStore struct {
	tokens   session.Tokens
	found    bool
	loadErr  error
	saveErr  error
	clearErr error

	loadCalls  int
	saveCalls  int
	clearCalls int
	lastSaved  session.Tokens
}

func (f *fakeStore) Load(_ context.Context) (session.Tokens, bool, error) {
	f.loadCalls++
	if f.loadErr != nil {
		return session.Tokens{}, false, f.loadErr
	}
	return f.tokens, f.found, nil
}

func (f *fakeStore) Save(_ context.Context, t session.Tokens) error {
	f.saveCalls++
	if f.saveErr != nil {
		return f.saveErr
	}
	f.tokens = t
	f.found = true
	f.lastSaved = t
	return nil
}

func (f *fakeStore) Clear(_ context.Context) error {
	f.clearCalls++
	if f.clearErr != nil {
		return f.clearErr
	}
	f.tokens = session.Tokens{}
	f.found = false
	return nil
}

// fakePrompt is an in-memory session.Prompt for unit tests.
type fakePrompt struct {
	email    string
	password string
	err      error

	calls       int
	lastDefault string
}

func (f *fakePrompt) Credentials(_ context.Context, defaultEmail string) (string, string, error) {
	f.calls++
	f.lastDefault = defaultEmail
	if f.err != nil {
		return "", "", f.err
	}
	return f.email, f.password, nil
}

// makeTokens builds a Tokens value with plausible test strings and the
// supplied expiries. Useful for arranging store state.
func makeTokens(accessExpiry, refreshExpiry time.Time) session.Tokens {
	return session.Tokens{
		AccessToken:           "access-" + accessExpiry.Format(time.RFC3339),
		RefreshToken:          "refresh-" + refreshExpiry.Format(time.RFC3339),
		AccessTokenExpiresAt:  accessExpiry,
		RefreshTokenExpiresAt: refreshExpiry,
		UserEmail:             "alice@example.com",
	}
}
