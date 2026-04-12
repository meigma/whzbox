package awsverify

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/meigma/whzbox/internal/core/sandbox"
)

// mockSTS is a test double for the stsClient interface. Each call
// pops the next scripted result from results until empty.
type mockSTS struct {
	results []mockResult
	calls   int
}

type mockResult struct {
	out *sts.GetCallerIdentityOutput
	err error
}

func (m *mockSTS) GetCallerIdentity(
	_ context.Context,
	_ *sts.GetCallerIdentityInput,
	_ ...func(*sts.Options),
) (*sts.GetCallerIdentityOutput, error) {
	r := m.results[0]
	m.results = m.results[1:]
	m.calls++
	return r.out, r.err
}

// newTestVerifier returns an STSVerifier whose client factory is hard-
// wired to the supplied mock. Backoff is shortened so retry tests
// finish in milliseconds.
func newTestVerifier(mock *mockSTS) *STSVerifier {
	v := New("us-east-1")
	v.retryBackoff = time.Millisecond
	v.clientFactory = func(_ context.Context, _ sandbox.Credentials, _ string) (stsClient, error) {
		return mock, nil
	}
	return v
}

func okResult() mockResult {
	return mockResult{
		out: &sts.GetCallerIdentityOutput{
			Account: new("123456789012"),
			UserId:  new("AIDATEST"),
			Arn:     new("arn:aws:iam::123456789012:user/test"),
		},
	}
}

func TestSTSVerifier_KindAndSlug(t *testing.T) {
	v := New("")
	if v.Kind() != sandbox.KindAWS {
		t.Errorf("Kind: got %v, want %v", v.Kind(), sandbox.KindAWS)
	}
	if v.Slug() != "aws-sandbox" {
		t.Errorf("Slug: got %q", v.Slug())
	}
	if v.region != defaultRegion {
		t.Errorf("empty region should default to %q, got %q", defaultRegion, v.region)
	}
}

func TestSTSVerifier_HappyPath(t *testing.T) {
	mock := &mockSTS{results: []mockResult{okResult()}}
	v := newTestVerifier(mock)

	id, err := v.VerifyCredentials(context.Background(), sandbox.Credentials{
		AccessKey: "AKIA",
		SecretKey: "secret",
	})
	if err != nil {
		t.Fatalf("VerifyCredentials: %v", err)
	}
	if id.Account != "123456789012" {
		t.Errorf("Account: got %q", id.Account)
	}
	if id.UserID != "AIDATEST" {
		t.Errorf("UserID: got %q", id.UserID)
	}
	if id.ARN != "arn:aws:iam::123456789012:user/test" {
		t.Errorf("ARN: got %q", id.ARN)
	}
	if id.Region != "us-east-1" {
		t.Errorf("Region: got %q", id.Region)
	}
	if mock.calls != 1 {
		t.Errorf("expected 1 call, got %d", mock.calls)
	}
}

func TestSTSVerifier_RetriesOnInvalidClientTokenId(t *testing.T) {
	// First two calls fail with the IAM propagation error, third succeeds.
	retryErr := errors.New("operation error sts GetCallerIdentity: api error InvalidClientTokenId")
	mock := &mockSTS{results: []mockResult{
		{err: retryErr},
		{err: retryErr},
		okResult(),
	}}
	v := newTestVerifier(mock)

	id, err := v.VerifyCredentials(context.Background(), sandbox.Credentials{AccessKey: "AKIA", SecretKey: "s"})
	if err != nil {
		t.Fatalf("VerifyCredentials: %v", err)
	}
	if id.Account == "" {
		t.Error("expected identity populated after retry")
	}
	if mock.calls != 3 {
		t.Errorf("expected 3 calls after retry, got %d", mock.calls)
	}
}

func TestSTSVerifier_NonRetryableError(t *testing.T) {
	mock := &mockSTS{results: []mockResult{
		{err: errors.New("api error AccessDenied: forbidden")},
	}}
	v := newTestVerifier(mock)

	_, err := v.VerifyCredentials(context.Background(), sandbox.Credentials{AccessKey: "A", SecretKey: "s"})
	if err == nil {
		t.Fatal("expected error")
	}
	if mock.calls != 1 {
		t.Errorf("non-retryable errors must not retry: got %d calls", mock.calls)
	}
}

func TestSTSVerifier_MaxAttemptsExhausted(t *testing.T) {
	retryErr := errors.New("InvalidClientTokenId")
	results := make([]mockResult, 15)
	for i := range results {
		results[i] = mockResult{err: retryErr}
	}
	mock := &mockSTS{results: results}
	v := newTestVerifier(mock)

	_, err := v.VerifyCredentials(context.Background(), sandbox.Credentials{AccessKey: "A", SecretKey: "s"})
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if mock.calls != 15 {
		t.Errorf("expected 15 calls, got %d", mock.calls)
	}
}

func TestSTSVerifier_ContextCancelledDuringRetryBackoff(t *testing.T) {
	retryErr := errors.New("InvalidClientTokenId")
	mock := &mockSTS{results: []mockResult{
		{err: retryErr},
		{err: retryErr},
	}}
	v := newTestVerifier(mock)
	// Make backoff long enough that we can cancel mid-wait.
	v.retryBackoff = 500 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := v.VerifyCredentials(ctx, sandbox.Credentials{AccessKey: "A", SecretKey: "s"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error: got %v, want context.Canceled", err)
	}
}

func TestSTSVerifier_ClientFactoryError(t *testing.T) {
	bang := errors.New("cannot build aws config")
	v := New("us-east-1")
	v.clientFactory = func(_ context.Context, _ sandbox.Credentials, _ string) (stsClient, error) {
		return nil, bang
	}
	_, err := v.VerifyCredentials(context.Background(), sandbox.Credentials{})
	if !errors.Is(err, bang) {
		t.Errorf("error: got %v, want %v", err, bang)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := map[string]bool{
		"InvalidClientTokenId: ...":                       true,
		"operation error STS ... InvalidClientTokenId: x": true,
		"api error AccessDenied: forbidden":               false,
		"context deadline exceeded":                       false,
		"":                                                false,
	}
	for in, want := range tests {
		var err error
		if in != "" {
			err = errors.New(in)
		}
		if got := isRetryable(err); got != want {
			t.Errorf("isRetryable(%q) = %v, want %v", in, got, want)
		}
	}
}
