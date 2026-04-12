package sandbox_test

import (
	"context"
	"time"

	"github.com/meigma/whzbox/internal/core/sandbox"
	"github.com/meigma/whzbox/internal/core/session"
)

// fakeAuth is an in-memory sandbox.SessionAuthorizer for unit tests.
type fakeAuth struct {
	tokens session.Tokens
	err    error

	calls int
}

func (f *fakeAuth) EnsureValid(_ context.Context) (session.Tokens, error) {
	f.calls++
	return f.tokens, f.err
}

// fakeManager is an in-memory sandbox.Manager for unit tests. Each
// call is counted and the most-recent arguments are captured for
// assertion by tests.
type fakeManager struct {
	createResult *sandbox.Sandbox
	createErr    error

	commitErr error

	destroyErr error

	activeResult *sandbox.Sandbox
	activeErr    error

	createCalls  int
	commitCalls  int
	destroyCalls int
	activeCalls  int

	lastCreateSlug     string
	lastCreateDuration time.Duration
	lastCommitSlug     string
	lastCommitDuration time.Duration
}

func (f *fakeManager) Create(
	_ context.Context,
	_ session.Tokens,
	slug string,
	d time.Duration,
) (*sandbox.Sandbox, error) {
	f.createCalls++
	f.lastCreateSlug = slug
	f.lastCreateDuration = d
	return f.createResult, f.createErr
}

func (f *fakeManager) Commit(_ context.Context, _ session.Tokens, slug string, d time.Duration) error {
	f.commitCalls++
	f.lastCommitSlug = slug
	f.lastCommitDuration = d
	return f.commitErr
}

func (f *fakeManager) Destroy(_ context.Context, _ session.Tokens) error {
	f.destroyCalls++
	return f.destroyErr
}

func (f *fakeManager) Active(_ context.Context, _ session.Tokens) (*sandbox.Sandbox, error) {
	f.activeCalls++
	return f.activeResult, f.activeErr
}

// fakeProvider is an in-memory sandbox.Provider for unit tests.
type fakeProvider struct {
	kind sandbox.Kind
	slug string

	verifyResult sandbox.Identity
	verifyErr    error

	verifyCalls int
}

func (f *fakeProvider) Kind() sandbox.Kind { return f.kind }
func (f *fakeProvider) Slug() string       { return f.slug }

func (f *fakeProvider) VerifyCredentials(_ context.Context, _ sandbox.Credentials) (sandbox.Identity, error) {
	f.verifyCalls++
	return f.verifyResult, f.verifyErr
}

// fakeStore is an in-memory sandbox.Store for unit tests.
type fakeStore struct {
	loaded    map[sandbox.Kind]*sandbox.Sandbox
	loadErr   error
	saveErr   error
	clearErr  error
	saved     *sandbox.Sandbox
	saveCalls int
	clearAll  int
}

func (f *fakeStore) Load(_ context.Context, kind sandbox.Kind) (*sandbox.Sandbox, bool, error) {
	if f.loadErr != nil {
		return nil, false, f.loadErr
	}
	sb, ok := f.loaded[kind]
	return sb, ok, nil
}

func (f *fakeStore) Save(_ context.Context, sb *sandbox.Sandbox) error {
	f.saveCalls++
	f.saved = sb
	return f.saveErr
}

func (f *fakeStore) ClearAll(_ context.Context) error {
	f.clearAll++
	return f.clearErr
}

// sampleSandbox builds a Sandbox that looks like what a real Manager
// would return from Create (credentials + console, no identity).
func sampleSandbox() *sandbox.Sandbox {
	return &sandbox.Sandbox{
		Credentials: sandbox.Credentials{
			AccessKey: "AKIA...TESTING",
			SecretKey: "secretsauce",
		},
		Console: sandbox.Console{
			URL:      "https://999999999999.signin.aws.amazon.com/console",
			Username: "Whiz_User_test",
			Password: "uuid-goes-here",
		},
		StartedAt: time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
		ExpiresAt: time.Date(2026, 4, 11, 13, 0, 0, 0, time.UTC),
	}
}

// sampleIdentity is what a successful VerifyCredentials returns.
func sampleIdentity() sandbox.Identity {
	return sandbox.Identity{
		Account: "999999999999",
		UserID:  "AIDAXYZ",
		ARN:     "arn:aws:iam::999999999999:user/Whiz_User_test",
		Region:  "us-east-1",
	}
}
