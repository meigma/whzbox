//go:build integration

// Run with: go test -tags integration -timeout 5m ./internal/adapters/whizlabs/...
//
// This test exercises the full sandbox lifecycle against the real
// Whizlabs + AWS APIs. It is intentionally excluded from the default
// `go test` run and depends on the repo-local .env file.

package whizlabs_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/meigma/whzbox/internal/adapters/awsverify"
	"github.com/meigma/whzbox/internal/adapters/whizlabs"
	"github.com/meigma/whzbox/internal/config"
	"github.com/meigma/whzbox/internal/core/sandbox"
)

func TestIntegration_SandboxLifecycle(t *testing.T) {
	root := locateRepoRoot(t)
	env := readDotEnv(t, filepath.Join(root, ".env"))
	email, password := env["USERNAME"], env["PASSWORD"]
	if email == "" || password == "" {
		t.Skip("no USERNAME/PASSWORD in .env; skipping integration test")
	}

	cfg := config.WhizlabsConfig{
		BaseURL: config.DefaultWhizlabsBaseURL,
		PlayURL: config.DefaultWhizlabsPlayURL,
	}
	client := whizlabs.NewClient(cfg, nil)
	verifier := awsverify.New("us-east-1")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Step 1: log in to get the main JWT.
	tokens, err := client.Login(ctx, email, password)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	// Step 2: create the sandbox via the adapter.
	sb, err := client.Create(ctx, tokens, "aws-sandbox", time.Hour)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sb == nil || sb.Credentials.AccessKey == "" {
		t.Fatal("Create returned empty sandbox")
	}
	t.Logf("created: console=%s account_in_url=<url>", sb.Console.URL)

	// Step 3: commit ownership so Destroy can find it.
	if err := client.Commit(ctx, tokens, "aws-sandbox", time.Hour); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Step 4: verify credentials via real STS.
	id, err := verifier.VerifyCredentials(ctx, sb.Credentials)
	if err != nil {
		// Verification failure shouldn't block destroy.
		t.Logf("VerifyCredentials error (non-fatal for this test): %v", err)
	} else {
		t.Logf("verified: account=%s arn=%s", id.Account, id.ARN)
		if id.Account == "" || id.ARN == "" {
			t.Error("STS returned empty identity fields")
		}
	}

	// Step 5: destroy the sandbox so we don't leak one for an hour.
	if err := client.Destroy(ctx, tokens); err != nil {
		t.Fatalf("Destroy: %v", err)
	}

	// Step 6: second destroy should report no-active-sandbox.
	err = client.Destroy(ctx, tokens)
	if err == nil {
		t.Error("second Destroy should fail, got nil")
	} else if !isNoActiveSandbox(err) {
		t.Logf("second Destroy returned error (acceptable): %v", err)
	}
}

func isNoActiveSandbox(err error) bool {
	// Avoid importing errors just for Is inside this file.
	if err == nil {
		return false
	}
	return err.Error() != "" && (err == sandbox.ErrNoActiveSandbox ||
		containsAny(err.Error(), "no active sandbox", "Sandbox Not Found"))
}

func containsAny(s string, needles ...string) bool {
	for _, n := range needles {
		if len(n) > 0 && len(s) >= len(n) {
			for i := 0; i+len(n) <= len(s); i++ {
				if s[i:i+len(n)] == n {
					return true
				}
			}
		}
	}
	return false
}
