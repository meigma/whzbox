package sandbox_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/meigma/whzbox/internal/core/clock"
	"github.com/meigma/whzbox/internal/core/sandbox"
	"github.com/meigma/whzbox/internal/core/session"
)

// testRig bundles the four port fakes + a clock so each test can
// wire a Service with minimal boilerplate.
type testRig struct {
	auth     *fakeAuth
	manager  *fakeManager
	provider *fakeProvider
	store    *fakeStore
	clock    *clock.Fake
	svc      *sandbox.Service
}

func newRig(t *testing.T) *testRig {
	t.Helper()
	prov := &fakeProvider{kind: sandbox.KindAWS, slug: "aws-sandbox"}
	auth := &fakeAuth{tokens: session.Tokens{AccessToken: "test-token"}}
	mgr := &fakeManager{}
	store := &fakeStore{}
	clk := &clock.Fake{T: time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)}
	svc := sandbox.NewService(
		auth,
		mgr,
		map[sandbox.Kind]sandbox.Provider{sandbox.KindAWS: prov},
		store,
		clk,
		nil,
	)
	return &testRig{auth: auth, manager: mgr, provider: prov, store: store, clock: clk, svc: svc}
}

// -----------------------------------------------------------------------------
// Create — happy path
// -----------------------------------------------------------------------------

func TestService_Create_HappyPath(t *testing.T) {
	r := newRig(t)
	r.manager.createResult = sampleSandbox()
	r.provider.verifyResult = sampleIdentity()

	got, err := r.svc.Create(context.Background(), sandbox.KindAWS, time.Hour)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got == nil {
		t.Fatal("Create returned nil sandbox")
	}
	if got.Kind != sandbox.KindAWS {
		t.Errorf("Kind: got %q, want %q", got.Kind, sandbox.KindAWS)
	}
	if got.Slug != "aws-sandbox" {
		t.Errorf("Slug: got %q, want %q", got.Slug, "aws-sandbox")
	}
	if got.Identity.Account != "999999999999" {
		t.Errorf("Identity.Account missing or wrong: %+v", got.Identity)
	}
	if got.Credentials.AccessKey == "" {
		t.Error("Credentials.AccessKey missing")
	}

	// All three ports should have been called exactly once.
	if r.auth.calls != 1 {
		t.Errorf("auth.EnsureValid calls: got %d, want 1", r.auth.calls)
	}
	if r.manager.createCalls != 1 {
		t.Errorf("manager.Create calls: got %d, want 1", r.manager.createCalls)
	}
	if r.manager.commitCalls != 1 {
		t.Errorf("manager.Commit calls: got %d, want 1", r.manager.commitCalls)
	}
	if r.manager.destroyCalls != 0 {
		t.Errorf("manager.Destroy should not be called on happy path: %d", r.manager.destroyCalls)
	}
	if r.provider.verifyCalls != 1 {
		t.Errorf("provider.VerifyCredentials calls: got %d, want 1", r.provider.verifyCalls)
	}

	// Create and Commit should have received the same slug + duration.
	if r.manager.lastCreateSlug != "aws-sandbox" || r.manager.lastCommitSlug != "aws-sandbox" {
		t.Errorf("slug mismatch: create=%q commit=%q", r.manager.lastCreateSlug, r.manager.lastCommitSlug)
	}
	if r.manager.lastCreateDuration != time.Hour || r.manager.lastCommitDuration != time.Hour {
		t.Errorf("duration mismatch: create=%v commit=%v", r.manager.lastCreateDuration, r.manager.lastCommitDuration)
	}
}

// -----------------------------------------------------------------------------
// Create — failure paths
// -----------------------------------------------------------------------------

func TestService_Create_UnknownKind(t *testing.T) {
	r := newRig(t)

	_, err := r.svc.Create(context.Background(), sandbox.Kind("gcp"), time.Hour)
	if !errors.Is(err, sandbox.ErrUnknownKind) {
		t.Errorf("error: got %v, want ErrUnknownKind", err)
	}
	// Must fail fast — no ports touched.
	if r.auth.calls != 0 || r.manager.createCalls != 0 {
		t.Errorf("unknown kind should not touch ports: auth=%d create=%d", r.auth.calls, r.manager.createCalls)
	}
}

func TestService_Create_AuthFails(t *testing.T) {
	r := newRig(t)
	bang := errors.New("session unavailable")
	r.auth.err = bang

	_, err := r.svc.Create(context.Background(), sandbox.KindAWS, time.Hour)
	if !errors.Is(err, bang) {
		t.Errorf("error: got %v, want wrapped %v", err, bang)
	}
	if r.manager.createCalls != 0 {
		t.Errorf("manager.Create should not be called when auth fails: %d", r.manager.createCalls)
	}
}

func TestService_Create_AuthReturnsPromptUnavailable(t *testing.T) {
	// Confirm session sentinels pass through cleanly — the CLI layer
	// relies on errors.Is for exit-code mapping.
	r := newRig(t)
	r.auth.err = session.ErrPromptUnavailable

	_, err := r.svc.Create(context.Background(), sandbox.KindAWS, time.Hour)
	if !errors.Is(err, session.ErrPromptUnavailable) {
		t.Errorf("error: got %v, want ErrPromptUnavailable", err)
	}
}

func TestService_Create_ManagerCreateFails(t *testing.T) {
	r := newRig(t)
	bang := errors.New("whizlabs 500")
	r.manager.createErr = bang

	_, err := r.svc.Create(context.Background(), sandbox.KindAWS, time.Hour)
	if !errors.Is(err, bang) {
		t.Errorf("error: got %v, want wrapped %v", err, bang)
	}
	if !errors.Is(err, sandbox.ErrProvider) {
		t.Errorf("error: got %v, want ErrProvider", err)
	}
	if r.manager.commitCalls != 0 {
		t.Errorf("commit should not run after create failure: %d", r.manager.commitCalls)
	}
	if r.manager.destroyCalls != 0 {
		t.Errorf("destroy should not run after create failure: %d", r.manager.destroyCalls)
	}
	if r.provider.verifyCalls != 0 {
		t.Errorf("verify should not run after create failure: %d", r.provider.verifyCalls)
	}
}

func TestService_Create_CommitFails_DestroyRollsBack(t *testing.T) {
	r := newRig(t)
	r.manager.createResult = sampleSandbox()
	bang := errors.New("commit rejected")
	r.manager.commitErr = bang

	_, err := r.svc.Create(context.Background(), sandbox.KindAWS, time.Hour)
	if !errors.Is(err, bang) {
		t.Errorf("error: got %v, want wrapped %v", err, bang)
	}
	if !errors.Is(err, sandbox.ErrProvider) {
		t.Errorf("error: got %v, want ErrProvider", err)
	}
	if r.manager.destroyCalls != 1 {
		t.Errorf("destroy should be called as rollback: got %d, want 1", r.manager.destroyCalls)
	}
	if r.provider.verifyCalls != 0 {
		t.Errorf("verify should not run after commit failure: %d", r.provider.verifyCalls)
	}
}

func TestService_Create_CommitFails_DestroyAlsoFails(t *testing.T) {
	r := newRig(t)
	r.manager.createResult = sampleSandbox()
	commitBang := errors.New("commit rejected")
	destroyBang := errors.New("destroy also rejected")
	r.manager.commitErr = commitBang
	r.manager.destroyErr = destroyBang

	_, err := r.svc.Create(context.Background(), sandbox.KindAWS, time.Hour)
	// The commit error is the primary cause and what the caller
	// should see.
	if !errors.Is(err, commitBang) {
		t.Errorf("error: got %v, want wrapped %v", err, commitBang)
	}
	if !errors.Is(err, sandbox.ErrProvider) {
		t.Errorf("error: got %v, want ErrProvider", err)
	}
	// Secondary destroy error should NOT shadow the primary cause.
	if errors.Is(err, destroyBang) {
		t.Errorf("destroy error should not surface: %v", err)
	}
	if r.manager.destroyCalls != 1 {
		t.Errorf("destroy should still have been attempted: %d", r.manager.destroyCalls)
	}
}

func TestService_Create_VerifyFails_SandboxStillReturned(t *testing.T) {
	r := newRig(t)
	r.manager.createResult = sampleSandbox()
	bang := errors.New("InvalidClientTokenId")
	r.provider.verifyErr = bang

	got, err := r.svc.Create(context.Background(), sandbox.KindAWS, time.Hour)
	// Sandbox must still be returned.
	if got == nil {
		t.Fatal("sandbox should be returned even when verification fails")
	}
	if got.Credentials.AccessKey == "" {
		t.Error("returned sandbox should carry credentials")
	}
	// Error should be sentinel-wrapped.
	if !errors.Is(err, sandbox.ErrVerificationFailed) {
		t.Errorf("error: got %v, want ErrVerificationFailed", err)
	}
	// Underlying error should also be wrapped for diagnosis.
	if !errors.Is(err, bang) {
		t.Errorf("error: got %v, want underlying %v", err, bang)
	}
	// No destroy on verification failure — the sandbox is real and
	// the user may want the creds.
	if r.manager.destroyCalls != 0 {
		t.Errorf("destroy should NOT run on verification failure: %d", r.manager.destroyCalls)
	}
}

// -----------------------------------------------------------------------------
// Create — sandbox cache
// -----------------------------------------------------------------------------

func TestService_Create_CachesSandboxOnSuccess(t *testing.T) {
	r := newRig(t)
	r.manager.createResult = sampleSandbox()
	r.provider.verifyResult = sampleIdentity()

	got, err := r.svc.Create(context.Background(), sandbox.KindAWS, time.Hour)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if r.store.saveCalls != 1 {
		t.Errorf("store.Save calls: got %d, want 1", r.store.saveCalls)
	}
	if r.store.saved == nil || r.store.saved.Credentials.AccessKey != got.Credentials.AccessKey {
		t.Errorf("cached sandbox mismatch: %+v", r.store.saved)
	}
	if !r.store.saved.Verified {
		t.Error("cached sandbox should be marked verified on successful verification")
	}
	if r.store.saved.Identity.Account == "" {
		t.Error("cached sandbox should carry verified identity")
	}
}

func TestService_Create_CachesSandboxOnVerifyFailure(t *testing.T) {
	// Creds are real and already billed — cache them even if verify fails.
	r := newRig(t)
	r.manager.createResult = sampleSandbox()
	r.provider.verifyErr = errors.New("InvalidClientTokenId")

	_, err := r.svc.Create(context.Background(), sandbox.KindAWS, time.Hour)
	if !errors.Is(err, sandbox.ErrVerificationFailed) {
		t.Fatalf("Create: %v", err)
	}
	if r.store.saveCalls != 1 {
		t.Errorf("store.Save should be called even on verify failure: got %d", r.store.saveCalls)
	}
	if r.store.saved == nil || r.store.saved.Verified {
		t.Errorf("cached sandbox should remain unverified after failed verification: %+v", r.store.saved)
	}
}

func TestService_Create_ReusesCachedSandbox(t *testing.T) {
	r := newRig(t)
	cached := sampleSandbox()
	cached.Kind = sandbox.KindAWS
	cached.Slug = "aws-sandbox"
	cached.Verified = true
	cached.Identity = sampleIdentity()
	// ExpiresAt = clock.Now() + 1h (clock is at 2026-04-11 12:00 UTC, sandbox
	// expires at 13:00 UTC — still valid).
	r.store.loaded = map[sandbox.Kind]*sandbox.Sandbox{sandbox.KindAWS: cached}

	got, err := r.svc.Create(context.Background(), sandbox.KindAWS, time.Hour)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got != cached {
		t.Errorf("expected cached sandbox to be returned, got %+v", got)
	}
	// No upstream calls should have been made.
	if r.auth.calls != 0 {
		t.Errorf("auth should not be called on cache hit: %d", r.auth.calls)
	}
	if r.manager.createCalls != 0 {
		t.Errorf("manager.Create should not be called on cache hit: %d", r.manager.createCalls)
	}
	if r.provider.verifyCalls != 0 {
		t.Errorf("verify should not re-run on cache hit: %d", r.provider.verifyCalls)
	}
	if r.store.saveCalls != 0 {
		t.Errorf("store.Save should not be called on cache hit: %d", r.store.saveCalls)
	}
}

func TestService_Create_ReverifiesUnverifiedCachedSandbox(t *testing.T) {
	r := newRig(t)
	cached := sampleSandbox()
	cached.Kind = sandbox.KindAWS
	cached.Slug = "aws-sandbox"
	cached.Verified = false
	r.store.loaded = map[sandbox.Kind]*sandbox.Sandbox{sandbox.KindAWS: cached}
	r.provider.verifyResult = sampleIdentity()

	got, err := r.svc.Create(context.Background(), sandbox.KindAWS, time.Hour)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got != cached {
		t.Fatalf("expected cached sandbox, got %+v", got)
	}
	if !got.Verified {
		t.Error("cached sandbox should be marked verified after re-check")
	}
	if got.Identity.Account != "999999999999" {
		t.Errorf("identity not refreshed: %+v", got.Identity)
	}
	if r.auth.calls != 0 {
		t.Errorf("auth should not be called for cached sandbox re-check: %d", r.auth.calls)
	}
	if r.manager.createCalls != 0 {
		t.Errorf("manager.Create should not be called for cached sandbox re-check: %d", r.manager.createCalls)
	}
	if r.provider.verifyCalls != 1 {
		t.Errorf("verify should run once for unverified cache: %d", r.provider.verifyCalls)
	}
	if r.store.saveCalls != 1 {
		t.Errorf("store.Save should persist the upgraded cache entry: %d", r.store.saveCalls)
	}
}

func TestService_Create_ReverifyFailureStaysExplicit(t *testing.T) {
	r := newRig(t)
	cached := sampleSandbox()
	cached.Kind = sandbox.KindAWS
	cached.Slug = "aws-sandbox"
	cached.Verified = false
	r.store.loaded = map[sandbox.Kind]*sandbox.Sandbox{sandbox.KindAWS: cached}
	bang := errors.New("InvalidClientTokenId")
	r.provider.verifyErr = bang

	got, err := r.svc.Create(context.Background(), sandbox.KindAWS, time.Hour)
	if got != cached {
		t.Fatalf("expected cached sandbox, got %+v", got)
	}
	if !errors.Is(err, sandbox.ErrVerificationFailed) {
		t.Fatalf("error: got %v, want ErrVerificationFailed", err)
	}
	if !errors.Is(err, bang) {
		t.Errorf("error: got %v, want wrapped %v", err, bang)
	}
	if got.Verified {
		t.Error("cached sandbox should remain unverified after failed re-check")
	}
	if r.auth.calls != 0 {
		t.Errorf("auth should not be called for cached sandbox re-check: %d", r.auth.calls)
	}
	if r.manager.createCalls != 0 {
		t.Errorf("manager.Create should not be called after failed re-check: %d", r.manager.createCalls)
	}
	if r.provider.verifyCalls != 1 {
		t.Errorf("verify should run once for unverified cache: %d", r.provider.verifyCalls)
	}
	if r.store.saveCalls != 1 {
		t.Errorf("store.Save should persist the unverified cache entry: %d", r.store.saveCalls)
	}
}

func TestService_Create_IgnoresExpiredCache(t *testing.T) {
	r := newRig(t)
	cached := sampleSandbox()
	// Expired 1 minute ago relative to the fake clock.
	cached.ExpiresAt = r.clock.T.Add(-time.Minute)
	r.store.loaded = map[sandbox.Kind]*sandbox.Sandbox{sandbox.KindAWS: cached}
	r.manager.createResult = sampleSandbox()
	r.provider.verifyResult = sampleIdentity()

	_, err := r.svc.Create(context.Background(), sandbox.KindAWS, time.Hour)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Expired cache → full provision + overwrite save.
	if r.manager.createCalls != 1 {
		t.Errorf("manager.Create should run when cache is expired: %d", r.manager.createCalls)
	}
	if r.store.saveCalls != 1 {
		t.Errorf("fresh sandbox should be cached: %d saves", r.store.saveCalls)
	}
}

func TestService_Create_SaveErrorDoesNotMaskSuccess(t *testing.T) {
	r := newRig(t)
	r.manager.createResult = sampleSandbox()
	r.provider.verifyResult = sampleIdentity()
	r.store.saveErr = errors.New("disk full")

	got, err := r.svc.Create(context.Background(), sandbox.KindAWS, time.Hour)
	if err != nil {
		t.Errorf("Create should succeed despite cache save failure: %v", err)
	}
	if got == nil {
		t.Fatal("Create returned nil sandbox despite a real provision")
	}
}

func TestService_Create_LoadErrorFallsThroughToFresh(t *testing.T) {
	r := newRig(t)
	r.store.loadErr = errors.New("cache read failed")
	r.manager.createResult = sampleSandbox()
	r.provider.verifyResult = sampleIdentity()

	_, err := r.svc.Create(context.Background(), sandbox.KindAWS, time.Hour)
	if err != nil {
		t.Fatalf("Create should succeed on cache-load error: %v", err)
	}
	if r.manager.createCalls != 1 {
		t.Errorf("fresh provision expected when cache load errors: %d", r.manager.createCalls)
	}
}

// -----------------------------------------------------------------------------
// Destroy
// -----------------------------------------------------------------------------

func TestService_Destroy_HappyPath(t *testing.T) {
	r := newRig(t)

	if err := r.svc.Destroy(context.Background()); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
	if r.manager.destroyCalls != 1 {
		t.Errorf("destroy calls: got %d, want 1", r.manager.destroyCalls)
	}
	if r.store.clearAll != 1 {
		t.Errorf("store.ClearAll calls: got %d, want 1", r.store.clearAll)
	}
}

func TestService_Destroy_NoActiveSandbox_DoesNotClearCache(t *testing.T) {
	// If Whizlabs says "nothing to destroy", we shouldn't touch the
	// local cache either — the Destroy is effectively a no-op.
	r := newRig(t)
	r.manager.destroyErr = sandbox.ErrNoActiveSandbox

	_ = r.svc.Destroy(context.Background())
	if r.store.clearAll != 0 {
		t.Errorf("ClearAll should not run on ErrNoActiveSandbox: %d", r.store.clearAll)
	}
}

func TestService_Destroy_AuthFails(t *testing.T) {
	r := newRig(t)
	r.auth.err = session.ErrPromptUnavailable

	err := r.svc.Destroy(context.Background())
	if !errors.Is(err, session.ErrPromptUnavailable) {
		t.Errorf("error: got %v, want ErrPromptUnavailable", err)
	}
	if r.manager.destroyCalls != 0 {
		t.Errorf("destroy should not run when auth fails: %d", r.manager.destroyCalls)
	}
}

func TestService_Destroy_NoActiveSandbox(t *testing.T) {
	r := newRig(t)
	r.manager.destroyErr = sandbox.ErrNoActiveSandbox

	err := r.svc.Destroy(context.Background())
	if !errors.Is(err, sandbox.ErrNoActiveSandbox) {
		t.Errorf("error: got %v, want ErrNoActiveSandbox", err)
	}
}

func TestService_Destroy_ManagerGenericError(t *testing.T) {
	r := newRig(t)
	bang := errors.New("whizlabs 500")
	r.manager.destroyErr = bang

	err := r.svc.Destroy(context.Background())
	if !errors.Is(err, bang) {
		t.Errorf("error: got %v, want %v", err, bang)
	}
	if !errors.Is(err, sandbox.ErrProvider) {
		t.Errorf("error: got %v, want ErrProvider", err)
	}
}

// -----------------------------------------------------------------------------
// Status
// -----------------------------------------------------------------------------

func TestService_Status_HappyPath(t *testing.T) {
	r := newRig(t)
	r.manager.activeResult = sampleSandbox()

	got, err := r.svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if got == nil {
		t.Fatal("Status returned nil sandbox")
	}
	if r.provider.verifyCalls != 0 {
		t.Errorf("Status must not re-verify credentials: %d verify calls", r.provider.verifyCalls)
	}
}

func TestService_Status_NoActive(t *testing.T) {
	r := newRig(t)
	r.manager.activeErr = sandbox.ErrNoActiveSandbox

	got, err := r.svc.Status(context.Background())
	if err != nil {
		t.Errorf("Status should return (nil, nil), got err: %v", err)
	}
	if got != nil {
		t.Errorf("Status should return nil sandbox when none active, got: %+v", got)
	}
}

func TestService_Status_NoActive_Wrapped(t *testing.T) {
	// The Manager adapter may wrap ErrNoActiveSandbox with context
	// (e.g., fmt.Errorf("query: %w", ...)); Status must still
	// translate it to (nil, nil).
	r := newRig(t)
	r.manager.activeErr = errors.New("something: " + sandbox.ErrNoActiveSandbox.Error())

	_, err := r.svc.Status(context.Background())
	// This is a plain errors.New, so errors.Is won't match — the
	// translation is only for explicitly wrapped errors. Confirm the
	// error surfaces (not silently swallowed).
	if err == nil {
		t.Error("unwrapped error should surface, not be silently swallowed")
	}
}

func TestService_Status_NoActive_ErrorsWrap(t *testing.T) {
	// With fmt.Errorf %w wrapping, the translation should work.
	r := newRig(t)
	r.manager.activeErr = errors.Join(errors.New("upstream"), sandbox.ErrNoActiveSandbox)

	got, err := r.svc.Status(context.Background())
	if err != nil {
		t.Errorf("wrapped ErrNoActiveSandbox should translate to (nil, nil), got: %v", err)
	}
	if got != nil {
		t.Errorf("wrapped ErrNoActiveSandbox should translate to nil sandbox")
	}
}

func TestService_Status_AuthFails(t *testing.T) {
	r := newRig(t)
	r.auth.err = session.ErrPromptUnavailable

	_, err := r.svc.Status(context.Background())
	if !errors.Is(err, session.ErrPromptUnavailable) {
		t.Errorf("error: got %v, want ErrPromptUnavailable", err)
	}
}

func TestService_Status_ManagerGenericError(t *testing.T) {
	r := newRig(t)
	bang := errors.New("whizlabs hung up")
	r.manager.activeErr = bang

	_, err := r.svc.Status(context.Background())
	if !errors.Is(err, bang) {
		t.Errorf("error: got %v, want %v", err, bang)
	}
	if !errors.Is(err, sandbox.ErrProvider) {
		t.Errorf("error: got %v, want ErrProvider", err)
	}
}
