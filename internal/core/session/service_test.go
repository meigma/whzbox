package session_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/meigma/whzbox/internal/core/clock"
	"github.com/meigma/whzbox/internal/core/session"
)

// testRig bundles the four port fakes + a clock so each test can wire a
// Service with minimal boilerplate. Callers mutate the public fields of
// the fakes before calling EnsureValid/Login/Logout.
type testRig struct {
	clock    *clock.Fake
	store    *fakeStore
	provider *fakeProvider
	prompt   *fakePrompt
	svc      *session.Service
}

func newRig(t *testing.T) *testRig {
	t.Helper()
	clk := &clock.Fake{T: baseTime}
	store := &fakeStore{}
	provider := &fakeProvider{}
	prompt := &fakePrompt{}
	svc := session.NewService(provider, store, prompt, clk, nil)
	return &testRig{clock: clk, store: store, provider: provider, prompt: prompt, svc: svc}
}

// freshTokens builds a Tokens that is well inside the refresh window at
// baseTime (1h access, 7d refresh).
func freshTokens() session.Tokens {
	return makeTokens(baseTime.Add(time.Hour), baseTime.Add(7*24*time.Hour))
}

// nearExpiryTokens builds a Tokens whose access token expires inside the
// refresh window (5m) but whose refresh token is still valid for days.
func nearExpiryTokens() session.Tokens {
	return makeTokens(baseTime.Add(5*time.Minute), baseTime.Add(7*24*time.Hour))
}

// expiredAccessTokens has an expired access token and a valid refresh
// token.
func expiredAccessTokens() session.Tokens {
	return makeTokens(baseTime.Add(-time.Hour), baseTime.Add(7*24*time.Hour))
}

// fullyExpiredTokens has both access and refresh tokens expired.
func fullyExpiredTokens() session.Tokens {
	return makeTokens(baseTime.Add(-time.Hour), baseTime.Add(-time.Minute))
}

// -----------------------------------------------------------------------------
// EnsureValid — cache hit paths
// -----------------------------------------------------------------------------

func TestService_EnsureValid_CacheHitFresh(t *testing.T) {
	r := newRig(t)
	r.store.tokens = freshTokens()
	r.store.found = true

	got, err := r.svc.EnsureValid(context.Background())
	if err != nil {
		t.Fatalf("EnsureValid: %v", err)
	}
	if got.AccessToken != r.store.tokens.AccessToken {
		t.Errorf("returned wrong tokens: got %q, want %q", got.AccessToken, r.store.tokens.AccessToken)
	}
	if r.provider.refreshCalls != 0 {
		t.Errorf("refresh should not be called: %d calls", r.provider.refreshCalls)
	}
	if r.provider.loginCalls != 0 {
		t.Errorf("login should not be called: %d calls", r.provider.loginCalls)
	}
	if r.prompt.calls != 0 {
		t.Errorf("prompt should not be called: %d calls", r.prompt.calls)
	}
	if r.store.saveCalls != 0 {
		t.Errorf("store.Save should not be called: %d calls", r.store.saveCalls)
	}
}

func TestService_EnsureValid_CacheHitNearExpiry_RefreshOK(t *testing.T) {
	r := newRig(t)
	r.store.tokens = nearExpiryTokens()
	r.store.found = true

	refreshed := makeTokens(baseTime.Add(15*time.Hour), baseTime.Add(7*24*time.Hour))
	r.provider.refreshResult = refreshed

	got, err := r.svc.EnsureValid(context.Background())
	if err != nil {
		t.Fatalf("EnsureValid: %v", err)
	}
	if got.AccessToken != refreshed.AccessToken {
		t.Errorf("returned stale tokens: got %q, want %q", got.AccessToken, refreshed.AccessToken)
	}
	if r.provider.refreshCalls != 1 {
		t.Errorf("refresh calls: got %d, want 1", r.provider.refreshCalls)
	}
	if r.provider.loginCalls != 0 {
		t.Errorf("login should not be called: %d calls", r.provider.loginCalls)
	}
	if r.store.lastSaved.AccessToken != refreshed.AccessToken {
		t.Errorf("store.Save did not receive refreshed tokens")
	}
	if r.prompt.calls != 0 {
		t.Errorf("prompt should not be called: %d calls", r.prompt.calls)
	}
}

func TestService_EnsureValid_CacheHitExpired_RefreshOK(t *testing.T) {
	r := newRig(t)
	r.store.tokens = expiredAccessTokens()
	r.store.found = true

	refreshed := makeTokens(baseTime.Add(15*time.Hour), baseTime.Add(7*24*time.Hour))
	r.provider.refreshResult = refreshed

	got, err := r.svc.EnsureValid(context.Background())
	if err != nil {
		t.Fatalf("EnsureValid: %v", err)
	}
	if got.AccessToken != refreshed.AccessToken {
		t.Errorf("returned wrong tokens")
	}
	if r.provider.refreshCalls != 1 {
		t.Errorf("refresh calls: got %d, want 1", r.provider.refreshCalls)
	}
}

// -----------------------------------------------------------------------------
// EnsureValid — refresh failure fallbacks
// -----------------------------------------------------------------------------

func TestService_EnsureValid_RefreshFails_PromptOK(t *testing.T) {
	r := newRig(t)
	r.store.tokens = nearExpiryTokens()
	r.store.found = true

	r.provider.refreshErr = session.ErrSessionExpired

	r.prompt.email = "new@example.com"
	r.prompt.password = "hunter2"

	fresh := makeTokens(baseTime.Add(15*time.Hour), baseTime.Add(7*24*time.Hour))
	r.provider.loginResult = fresh

	got, err := r.svc.EnsureValid(context.Background())
	if err != nil {
		t.Fatalf("EnsureValid: %v", err)
	}
	if got.AccessToken != fresh.AccessToken {
		t.Errorf("returned wrong tokens")
	}
	if r.provider.refreshCalls != 1 {
		t.Errorf("refresh calls: got %d, want 1", r.provider.refreshCalls)
	}
	if r.provider.loginCalls != 1 {
		t.Errorf("login calls: got %d, want 1", r.provider.loginCalls)
	}
	if r.provider.lastEmail != "new@example.com" {
		t.Errorf("lastEmail: got %q, want %q", r.provider.lastEmail, "new@example.com")
	}
	if r.store.lastSaved.AccessToken != fresh.AccessToken {
		t.Errorf("store.Save did not receive fresh tokens")
	}
}

func TestService_EnsureValid_RefreshFails_NoPromptAvailable(t *testing.T) {
	r := newRig(t)
	r.store.tokens = nearExpiryTokens()
	r.store.found = true

	r.provider.refreshErr = session.ErrSessionExpired
	r.prompt.err = session.ErrPromptUnavailable

	_, err := r.svc.EnsureValid(context.Background())
	if !errors.Is(err, session.ErrPromptUnavailable) {
		t.Errorf("error: got %v, want ErrPromptUnavailable", err)
	}
	if r.provider.loginCalls != 0 {
		t.Errorf("login should not be called: %d", r.provider.loginCalls)
	}
	if r.store.saveCalls != 0 {
		t.Errorf("store.Save should not be called on error path: %d", r.store.saveCalls)
	}
}

func TestService_EnsureValid_NotRefreshable_PromptOK(t *testing.T) {
	r := newRig(t)
	r.store.tokens = fullyExpiredTokens()
	r.store.found = true

	r.prompt.email = "alice@example.com"
	r.prompt.password = "pw"

	fresh := freshTokens()
	r.provider.loginResult = fresh

	got, err := r.svc.EnsureValid(context.Background())
	if err != nil {
		t.Fatalf("EnsureValid: %v", err)
	}
	if got.AccessToken != fresh.AccessToken {
		t.Errorf("returned wrong tokens")
	}
	if r.provider.refreshCalls != 0 {
		t.Errorf("refresh should not be called when not refreshable: %d", r.provider.refreshCalls)
	}
	if r.provider.loginCalls != 1 {
		t.Errorf("login calls: got %d, want 1", r.provider.loginCalls)
	}
}

// -----------------------------------------------------------------------------
// EnsureValid — cache miss
// -----------------------------------------------------------------------------

func TestService_EnsureValid_CacheMiss_PromptOK(t *testing.T) {
	r := newRig(t)
	// store default: found=false

	r.prompt.email = "alice@example.com"
	r.prompt.password = "pw"
	fresh := freshTokens()
	r.provider.loginResult = fresh

	got, err := r.svc.EnsureValid(context.Background())
	if err != nil {
		t.Fatalf("EnsureValid: %v", err)
	}
	if got.AccessToken != fresh.AccessToken {
		t.Errorf("returned wrong tokens")
	}
	if r.prompt.lastDefault != "" {
		t.Errorf("lastDefault email should be empty on cache miss, got %q", r.prompt.lastDefault)
	}
	if r.store.saveCalls != 1 {
		t.Errorf("store.Save calls: got %d, want 1", r.store.saveCalls)
	}
}

func TestService_EnsureValid_CacheMiss_NoPrompt(t *testing.T) {
	r := newRig(t)
	r.prompt.err = session.ErrPromptUnavailable

	_, err := r.svc.EnsureValid(context.Background())
	if !errors.Is(err, session.ErrPromptUnavailable) {
		t.Errorf("error: got %v, want ErrPromptUnavailable", err)
	}
}

// -----------------------------------------------------------------------------
// EnsureValid — error paths
// -----------------------------------------------------------------------------

func TestService_EnsureValid_UserAborts(t *testing.T) {
	r := newRig(t)
	r.prompt.err = session.ErrUserAborted

	_, err := r.svc.EnsureValid(context.Background())
	if !errors.Is(err, session.ErrUserAborted) {
		t.Errorf("error: got %v, want ErrUserAborted", err)
	}
}

func TestService_EnsureValid_InvalidCredentials(t *testing.T) {
	r := newRig(t)
	r.prompt.email = "alice@example.com"
	r.prompt.password = "wrong"
	r.provider.loginErr = session.ErrInvalidCredentials

	_, err := r.svc.EnsureValid(context.Background())
	if !errors.Is(err, session.ErrInvalidCredentials) {
		t.Errorf("error: got %v, want ErrInvalidCredentials", err)
	}
	if r.store.saveCalls != 0 {
		t.Errorf("store.Save should not be called on invalid credentials: %d", r.store.saveCalls)
	}
}

func TestService_EnsureValid_StoreLoadError(t *testing.T) {
	r := newRig(t)
	bang := errors.New("disk on fire")
	r.store.loadErr = bang

	_, err := r.svc.EnsureValid(context.Background())
	if err == nil || !errors.Is(err, bang) {
		t.Errorf("error: got %v, want wrapped %v", err, bang)
	}
}

func TestService_EnsureValid_StoreSaveErrorAfterRefresh(t *testing.T) {
	r := newRig(t)
	r.store.tokens = nearExpiryTokens()
	r.store.found = true

	r.provider.refreshResult = makeTokens(baseTime.Add(15*time.Hour), baseTime.Add(7*24*time.Hour))
	bang := errors.New("disk full")
	r.store.saveErr = bang

	_, err := r.svc.EnsureValid(context.Background())
	if err == nil || !errors.Is(err, bang) {
		t.Errorf("error: got %v, want wrapped %v", err, bang)
	}
}

func TestService_EnsureValid_StoreSaveErrorAfterLogin(t *testing.T) {
	r := newRig(t)
	r.prompt.email = "alice@example.com"
	r.prompt.password = "pw"
	r.provider.loginResult = freshTokens()
	bang := errors.New("read-only fs")
	r.store.saveErr = bang

	_, err := r.svc.EnsureValid(context.Background())
	if err == nil || !errors.Is(err, bang) {
		t.Errorf("error: got %v, want wrapped %v", err, bang)
	}
}

func TestService_EnsureValid_WrapsGenericPromptError(t *testing.T) {
	r := newRig(t)
	bang := errors.New("terminal broke")
	r.prompt.err = bang

	_, err := r.svc.EnsureValid(context.Background())
	if err == nil || !errors.Is(err, bang) {
		t.Errorf("error: got %v, want wrapped %v", err, bang)
	}
	// Sentinel check: generic errors should NOT look like our sentinels.
	if errors.Is(err, session.ErrPromptUnavailable) {
		t.Error("generic error should not match ErrPromptUnavailable")
	}
	if errors.Is(err, session.ErrUserAborted) {
		t.Error("generic error should not match ErrUserAborted")
	}
}

func TestService_EnsureValid_WrapsGenericLoginError(t *testing.T) {
	r := newRig(t)
	r.prompt.email = "a@b"
	r.prompt.password = "c"
	bang := errors.New("server exploded")
	r.provider.loginErr = bang

	_, err := r.svc.EnsureValid(context.Background())
	if err == nil || !errors.Is(err, bang) {
		t.Errorf("error: got %v, want wrapped %v", err, bang)
	}
	if errors.Is(err, session.ErrInvalidCredentials) {
		t.Error("generic error should not match ErrInvalidCredentials")
	}
}

func TestService_EnsureValid_DefaultEmailHintFromCache(t *testing.T) {
	r := newRig(t)
	r.store.tokens = fullyExpiredTokens() // cached user but unusable
	r.store.found = true

	r.prompt.email = r.store.tokens.UserEmail
	r.prompt.password = "pw"
	r.provider.loginResult = freshTokens()

	if _, err := r.svc.EnsureValid(context.Background()); err != nil {
		t.Fatalf("EnsureValid: %v", err)
	}
	if r.prompt.lastDefault != r.store.tokens.UserEmail {
		t.Errorf("lastDefault: got %q, want %q", r.prompt.lastDefault, r.store.tokens.UserEmail)
	}
}

// -----------------------------------------------------------------------------
// Login — explicit
// -----------------------------------------------------------------------------

func TestService_Login_IgnoresFreshCache(t *testing.T) {
	r := newRig(t)
	r.store.tokens = freshTokens()
	r.store.found = true

	r.prompt.email = "new@example.com"
	r.prompt.password = "pw"
	replaced := makeTokens(baseTime.Add(15*time.Hour), baseTime.Add(7*24*time.Hour))
	r.provider.loginResult = replaced

	got, err := r.svc.Login(context.Background())
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if got.AccessToken != replaced.AccessToken {
		t.Errorf("returned wrong tokens")
	}
	if r.provider.refreshCalls != 0 {
		t.Error("Login should never refresh")
	}
	if r.provider.loginCalls != 1 {
		t.Errorf("login calls: got %d, want 1", r.provider.loginCalls)
	}
	if r.store.lastSaved.AccessToken != replaced.AccessToken {
		t.Errorf("store.Save did not receive replaced tokens")
	}
}

func TestService_Login_UsesCachedEmailHint(t *testing.T) {
	r := newRig(t)
	r.store.tokens = freshTokens() // has UserEmail="alice@example.com"
	r.store.found = true

	r.prompt.email = "alice@example.com"
	r.prompt.password = "pw"
	r.provider.loginResult = freshTokens()

	if _, err := r.svc.Login(context.Background()); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if r.prompt.lastDefault != "alice@example.com" {
		t.Errorf("lastDefault: got %q, want %q", r.prompt.lastDefault, "alice@example.com")
	}
}

// -----------------------------------------------------------------------------
// Logout
// -----------------------------------------------------------------------------

func TestService_Logout_ClearsStore(t *testing.T) {
	r := newRig(t)
	r.store.tokens = freshTokens()
	r.store.found = true

	if err := r.svc.Logout(context.Background()); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if r.store.clearCalls != 1 {
		t.Errorf("clear calls: got %d, want 1", r.store.clearCalls)
	}
	if r.store.found {
		t.Error("store.found should be false after clear")
	}
}

func TestService_Logout_EmptyStoreIsNoop(t *testing.T) {
	r := newRig(t)
	// store default: found=false

	if err := r.svc.Logout(context.Background()); err != nil {
		t.Fatalf("Logout on empty store returned error: %v", err)
	}
	if r.store.clearCalls != 1 {
		t.Errorf("clear calls: got %d, want 1", r.store.clearCalls)
	}
}

func TestService_Logout_WrapsClearError(t *testing.T) {
	r := newRig(t)
	bang := errors.New("rm failed")
	r.store.clearErr = bang

	err := r.svc.Logout(context.Background())
	if err == nil || !errors.Is(err, bang) {
		t.Errorf("error: got %v, want wrapped %v", err, bang)
	}
}
