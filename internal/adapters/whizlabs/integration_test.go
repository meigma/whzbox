//go:build integration

// Run with: go test -tags integration ./internal/adapters/whizlabs/...
//
// This test talks to the real Whizlabs API using credentials from the
// repo-local .env file. It is NOT part of the default go test run.

package whizlabs_test

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/meigma/whzbox/internal/adapters/whizlabs"
	"github.com/meigma/whzbox/internal/adapters/xdgstore"
	"github.com/meigma/whzbox/internal/config"
	"github.com/meigma/whzbox/internal/core/clock"
	"github.com/meigma/whzbox/internal/core/session"
)

// readDotEnv is a minimal parser that understands KEY="value" and
// KEY=value lines. Good enough for a two-line credentials file.
func readDotEnv(t *testing.T, path string) map[string]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	out := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan %s: %v", path, err)
	}
	return out
}

// locateRepoRoot walks up from the test file until it finds go.mod.
func locateRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found walking up from %s", dir)
		}
		dir = parent
	}
}

func TestIntegration_FullLoginLifecycle(t *testing.T) {
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
	c := whizlabs.NewClient(cfg, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Step 1: real HTTP login.
	tokens, err := c.Login(ctx, email, password)
	if err != nil {
		t.Fatalf("whizlabs Login: %v", err)
	}
	if tokens.AccessToken == "" {
		t.Fatal("Login returned empty access token")
	}
	if tokens.UserEmail != email {
		t.Errorf("Login returned email %q, want %q", tokens.UserEmail, email)
	}
	if tokens.AccessTokenExpiresAt.Before(time.Now()) {
		t.Error("Login returned already-expired access token")
	}

	// Step 2: save to a tempdir store.
	storeDir := filepath.Join(t.TempDir(), "whzbox")
	store, err := xdgstore.New(storeDir, nil)
	if err != nil {
		t.Fatalf("xdgstore.New: %v", err)
	}
	if err := store.Save(ctx, tokens); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	// Step 3: round-trip load.
	loaded, found, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("store.Load: %v", err)
	}
	if !found {
		t.Fatal("store.Load: found=false after Save")
	}
	if loaded.AccessToken != tokens.AccessToken {
		t.Error("round-trip AccessToken mismatch")
	}

	// Step 4: refresh with the access token we just got. This confirms
	// the /auth/exchange endpoint accepts a fresh token — important
	// groundwork for silent refresh in the service.
	refreshed, err := c.Refresh(ctx, loaded)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if refreshed.AccessToken == "" {
		t.Fatal("Refresh returned empty access token")
	}
	if refreshed.UserEmail != email {
		t.Errorf("Refresh email: got %q, want %q", refreshed.UserEmail, email)
	}

	// Step 5: the session.Service happy path against real ports.
	// This exercises the exact code the `whzbox login` command runs.
	prompt := &scriptedPrompt{email: email, password: password}
	svc := session.NewService(c, store, prompt, clock.Real{}, nil)
	if err := store.Clear(ctx); err != nil {
		t.Fatalf("store.Clear: %v", err)
	}
	svcTokens, err := svc.Login(ctx)
	if err != nil {
		t.Fatalf("svc.Login: %v", err)
	}
	if svcTokens.AccessToken == "" {
		t.Fatal("svc.Login returned empty tokens")
	}

	// Step 6: logout clears the store.
	if err := svc.Logout(ctx); err != nil {
		t.Fatalf("svc.Logout: %v", err)
	}
	if _, found, _ := store.Load(ctx); found {
		t.Error("store should be empty after Logout")
	}

	t.Logf("full lifecycle ok: user=%s access-exp=%s refresh-exp=%s",
		svcTokens.UserEmail,
		svcTokens.AccessTokenExpiresAt.Format(time.RFC3339),
		svcTokens.RefreshTokenExpiresAt.Format(time.RFC3339),
	)
}

// scriptedPrompt returns fixed credentials without touching a TTY.
type scriptedPrompt struct {
	email    string
	password string
}

func (s *scriptedPrompt) Credentials(_ context.Context, _ string) (string, string, error) {
	return s.email, s.password, nil
}
