package awsverify

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/meigma/whzbox/internal/core/sandbox"
)

// Default tuning constants. These match what the feasibility prototype
// found to work against a freshly-minted IAM user: roughly two retries
// over six seconds is enough for propagation in practice.
const (
	defaultRegion       = "us-east-1"
	defaultMaxAttempts  = 15
	defaultRetryBackoff = 3 * time.Second
)

// stsClient is the subset of sts.Client we actually call. Declaring it
// as an interface lets tests inject a mock without depending on the
// AWS SDK mock suite.
type stsClient interface {
	GetCallerIdentity(
		ctx context.Context,
		params *sts.GetCallerIdentityInput,
		optFns ...func(*sts.Options),
	) (*sts.GetCallerIdentityOutput, error)
}

// STSVerifier implements sandbox.Provider for KindAWS by calling
// sts:GetCallerIdentity with the sandbox credentials. On
// InvalidClientTokenId (IAM key still propagating) it retries with a
// fixed delay up to maxAttempts times; on any other error it returns
// immediately.
type STSVerifier struct {
	region       string
	maxAttempts  int
	retryBackoff time.Duration

	// clientFactory builds an stsClient from credentials. Production
	// uses newRealSTSClient; tests swap in a closure returning a
	// mockSTS to avoid touching the real AWS SDK.
	clientFactory func(ctx context.Context, creds sandbox.Credentials, region string) (stsClient, error)
}

// New returns an STSVerifier targeting the given region. An empty
// region falls back to us-east-1, which is where every Whizlabs AWS
// sandbox lives today.
func New(region string) *STSVerifier {
	if region == "" {
		region = defaultRegion
	}
	return &STSVerifier{
		region:        region,
		maxAttempts:   defaultMaxAttempts,
		retryBackoff:  defaultRetryBackoff,
		clientFactory: newRealSTSClient,
	}
}

// newRealSTSClient builds a real AWS SDK STS client from static
// credentials. It is kept as a package-level function (not a method)
// so tests can cleanly substitute it via the clientFactory field.
func newRealSTSClient(ctx context.Context, creds sandbox.Credentials, region string) (stsClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(creds.AccessKey, creds.SecretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	return sts.NewFromConfig(cfg), nil
}

// Kind implements sandbox.Provider.
func (v *STSVerifier) Kind() sandbox.Kind { return sandbox.KindAWS }

// Slug implements sandbox.Provider. Matches the Whizlabs upstream slug.
func (v *STSVerifier) Slug() string { return "aws-sandbox" }

// VerifyCredentials implements sandbox.Provider. It calls
// GetCallerIdentity and retries on IAM propagation errors.
//
// The caller's context governs the overall deadline; the retry loop
// yields to ctx.Done on every backoff so a canceled command does not
// hang for the full retry budget.
func (v *STSVerifier) VerifyCredentials(ctx context.Context, creds sandbox.Credentials) (sandbox.Identity, error) {
	client, err := v.clientFactory(ctx, creds, v.region)
	if err != nil {
		return sandbox.Identity{}, err
	}

	var (
		out     *sts.GetCallerIdentityOutput
		lastErr error
	)
	for attempt := 1; attempt <= v.maxAttempts; attempt++ {
		out, lastErr = client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if lastErr == nil {
			break
		}
		if !isRetryable(lastErr) || attempt == v.maxAttempts {
			return sandbox.Identity{}, lastErr
		}
		select {
		case <-ctx.Done():
			return sandbox.Identity{}, ctx.Err()
		case <-time.After(v.retryBackoff):
		}
	}

	return sandbox.Identity{
		Account: deref(out.Account),
		UserID:  deref(out.UserId),
		ARN:     deref(out.Arn),
		Region:  v.region,
	}, nil
}

// isRetryable reports whether an error from GetCallerIdentity represents
// IAM propagation delay (which is worth retrying) versus a permanent
// failure like AccessDenied (which is not).
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	// AWS smithy errors bubble up with the API error code in the
	// message. We only care about the specific "new key, try again
	// in a sec" code.
	return strings.Contains(err.Error(), "InvalidClientTokenId")
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
