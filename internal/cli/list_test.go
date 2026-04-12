package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/meigma/whzbox/internal/config"
	"github.com/meigma/whzbox/internal/core/clock"
	"github.com/meigma/whzbox/internal/core/sandbox"
)

// listSandboxStore is a sandbox.Store preloaded with a fixed set of
// sandboxes. Only Load/LoadAll are exercised by the list command.
type listSandboxStore struct {
	entries []*sandbox.Sandbox
}

func (s *listSandboxStore) Load(_ context.Context, kind sandbox.Kind) (*sandbox.Sandbox, bool, error) {
	for _, sb := range s.entries {
		if sb.Kind == kind {
			return sb, true, nil
		}
	}
	return nil, false, nil
}
func (s *listSandboxStore) LoadAll(_ context.Context) ([]*sandbox.Sandbox, error) {
	return s.entries, nil
}
func (s *listSandboxStore) Save(_ context.Context, _ *sandbox.Sandbox) error { return nil }
func (s *listSandboxStore) ClearAll(_ context.Context) error                 { return nil }

func newListTestApp(t *testing.T, now time.Time, entries ...*sandbox.Sandbox) *App {
	t.Helper()
	store := &listSandboxStore{entries: entries}
	ver := &stubVerifier{}
	svc := sandbox.NewService(
		&stubAuth{},
		&stubManager{},
		map[sandbox.Kind]sandbox.Provider{sandbox.KindAWS: ver},
		store,
		&clock.Fake{T: now},
		nil,
	)
	return &App{
		Sandbox: svc,
		Clock:   &clock.Fake{T: now},
		Config:  config.Config{},
	}
}

func sampleListEntry(account string, expires time.Time) *sandbox.Sandbox {
	return &sandbox.Sandbox{
		Kind: sandbox.KindAWS,
		Slug: "aws-sandbox",
		Credentials: sandbox.Credentials{
			AccessKey: "AKIA_" + account,
			SecretKey: "sec_" + account,
		},
		Identity: sandbox.Identity{
			Account: account,
			Region:  "us-east-1",
		},
		StartedAt: expires.Add(-time.Hour),
		ExpiresAt: expires,
	}
}

func TestListCommand_Empty(t *testing.T) {
	now := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)
	app := newListTestApp(t, now)

	cmd := newListCommand(&app)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out.String(), "(no sandboxes cached)") {
		t.Errorf("missing empty marker:\n%s", out.String())
	}
}

func TestListCommand_ActiveAndExpired(t *testing.T) {
	now := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)
	active := sampleListEntry("111111111111", now.Add(time.Hour))
	expired := sampleListEntry("222222222222", now.Add(-time.Hour))
	app := newListTestApp(t, now, active, expired)

	cmd := newListCommand(&app)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "111111111111") || !strings.Contains(got, "active") {
		t.Errorf("missing active row:\n%s", got)
	}
	if !strings.Contains(got, "222222222222") || !strings.Contains(got, "expired") {
		t.Errorf("missing expired row:\n%s", got)
	}
}

func TestListCommand_JSON(t *testing.T) {
	now := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)
	sb := sampleListEntry("111111111111", now.Add(time.Hour))
	app := newListTestApp(t, now, sb)
	app.Config.JSON = true

	cmd := newListCommand(&app)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var decoded []map[string]any
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, out.String())
	}
	if len(decoded) != 1 {
		t.Fatalf("want 1 entry, got %d", len(decoded))
	}
	if decoded[0]["kind"] != "aws" {
		t.Errorf("kind: got %v, want aws", decoded[0]["kind"])
	}
	id, _ := decoded[0]["identity"].(map[string]any)
	if id["account"] != "111111111111" {
		t.Errorf("account: got %v", id["account"])
	}
}

func TestListCommand_JSON_Empty(t *testing.T) {
	now := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)
	app := newListTestApp(t, now)
	app.Config.JSON = true

	cmd := newListCommand(&app)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != "[]" {
		t.Errorf("empty JSON: got %q, want %q", got, "[]")
	}
}
