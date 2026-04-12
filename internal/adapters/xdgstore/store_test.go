package xdgstore_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/meigma/whzbox/internal/adapters/xdgstore"
	"github.com/meigma/whzbox/internal/core/sandbox"
	"github.com/meigma/whzbox/internal/core/session"
)

func tempStore(t *testing.T) *xdgstore.Store {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "whzbox")
	s, err := xdgstore.New(dir, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

func sampleTokens() session.Tokens {
	return session.Tokens{
		AccessToken:           "access-abc",
		RefreshToken:          "refresh-def",
		AccessTokenExpiresAt:  time.Date(2026, 4, 11, 20, 0, 0, 0, time.UTC),
		RefreshTokenExpiresAt: time.Date(2026, 4, 18, 20, 0, 0, 0, time.UTC),
		UserEmail:             "alice@example.com",
	}
}

func TestStore_LoadMissing(t *testing.T) {
	s := tempStore(t)
	_, found, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if found {
		t.Error("found should be false when no file exists")
	}
}

func TestStore_SaveAndLoad(t *testing.T) {
	s := tempStore(t)
	orig := sampleTokens()

	if err := s.Save(context.Background(), orig); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, found, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !found {
		t.Fatal("Load returned found=false after Save")
	}
	if got.AccessToken != orig.AccessToken {
		t.Errorf("AccessToken mismatch")
	}
	if got.RefreshToken != orig.RefreshToken {
		t.Errorf("RefreshToken mismatch")
	}
	if !got.AccessTokenExpiresAt.Equal(orig.AccessTokenExpiresAt) {
		t.Errorf("AccessTokenExpiresAt: got %v, want %v", got.AccessTokenExpiresAt, orig.AccessTokenExpiresAt)
	}
	if !got.RefreshTokenExpiresAt.Equal(orig.RefreshTokenExpiresAt) {
		t.Errorf("RefreshTokenExpiresAt mismatch")
	}
	if got.UserEmail != orig.UserEmail {
		t.Errorf("UserEmail mismatch")
	}
}

func TestStore_FilePermsAfterSave(t *testing.T) {
	s := tempStore(t)
	if err := s.Save(context.Background(), sampleTokens()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(s.Path())
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if m := info.Mode().Perm(); m != 0o600 {
		t.Errorf("file mode: got %v, want 0600", m)
	}

	dirInfo, err := os.Stat(filepath.Dir(s.Path()))
	if err != nil {
		t.Fatalf("Stat dir: %v", err)
	}
	if m := dirInfo.Mode().Perm(); m != 0o700 {
		t.Errorf("dir mode: got %v, want 0700", m)
	}
}

func TestStore_WidePermsRejected(t *testing.T) {
	s := tempStore(t)
	if err := s.Save(context.Background(), sampleTokens()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := os.Chmod(s.Path(), 0o644); err != nil {
		t.Fatalf("Chmod: %v", err)
	}

	_, _, err := s.Load(context.Background())
	if err == nil {
		t.Error("Load should reject wide file perms, got nil")
	}
}

func TestStore_CorruptJSONSelfHeals(t *testing.T) {
	s := tempStore(t)
	if err := os.WriteFile(s.Path(), []byte("not json"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, found, err := s.Load(context.Background())
	if err != nil {
		t.Errorf("Load should self-heal corrupt files, got error: %v", err)
	}
	if found {
		t.Error("found should be false after corruption")
	}
	if _, statErr := os.Stat(s.Path()); !errors.Is(statErr, fs.ErrNotExist) {
		t.Error("corrupt file should be removed")
	}
}

func TestStore_UnknownSchemaVersionSelfHeals(t *testing.T) {
	s := tempStore(t)
	bad := `{"version":999,"whizlabs":{}}`
	if err := os.WriteFile(s.Path(), []byte(bad), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, found, err := s.Load(context.Background())
	if err != nil {
		t.Errorf("Load should self-heal unknown version, got error: %v", err)
	}
	if found {
		t.Error("found should be false for unknown version")
	}
}

func TestStore_UnparsableTimestampsSelfHeals(t *testing.T) {
	s := tempStore(t)
	bad := `{"version":1,"whizlabs":{"user_email":"a@b","access_token":"x","refresh_token":"y","access_token_expires_at":"not-a-time","refresh_token_expires_at":"also-not-a-time"}}`
	if err := os.WriteFile(s.Path(), []byte(bad), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, found, err := s.Load(context.Background())
	if err != nil {
		t.Errorf("Load should self-heal bad timestamps, got error: %v", err)
	}
	if found {
		t.Error("found should be false for unparsable timestamps")
	}
}

func TestStore_Clear(t *testing.T) {
	s := tempStore(t)
	if err := s.Save(context.Background(), sampleTokens()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := s.Clear(context.Background()); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if _, err := os.Stat(s.Path()); !errors.Is(err, fs.ErrNotExist) {
		t.Error("file should be removed after Clear")
	}
}

func TestStore_ClearMissingIsNoop(t *testing.T) {
	s := tempStore(t)
	if err := s.Clear(context.Background()); err != nil {
		t.Errorf("Clear on missing file should be a no-op, got: %v", err)
	}
}

func TestStore_SaveIsAtomic(t *testing.T) {
	s := tempStore(t)
	// Do a save so a real file exists.
	if err := s.Save(context.Background(), sampleTokens()); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	// After save the temp file should not linger.
	if _, err := os.Stat(s.Path() + ".tmp"); !errors.Is(err, fs.ErrNotExist) {
		t.Error("temp file should be gone after successful Save")
	}
}

// -----------------------------------------------------------------------------
// Sandbox cache
// -----------------------------------------------------------------------------

func sampleSandbox() *sandbox.Sandbox {
	return &sandbox.Sandbox{
		Kind:        sandbox.KindAWS,
		Slug:        "aws-sandbox",
		Credentials: sandbox.Credentials{AccessKey: "AKIA123", SecretKey: "secret"},
		Console:     sandbox.Console{URL: "https://acct.signin.aws/", Username: "whiz", Password: "pw"},
		Identity:    sandbox.Identity{Account: "111111111111", UserID: "AIDA", ARN: "arn:...", Region: "us-east-1"},
		StartedAt:   time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
		ExpiresAt:   time.Date(2026, 4, 11, 13, 0, 0, 0, time.UTC),
	}
}

func TestStore_SandboxRoundTrip(t *testing.T) {
	s := tempStore(t)
	orig := sampleSandbox()

	if err := s.SaveSandbox(context.Background(), orig); err != nil {
		t.Fatalf("SaveSandbox: %v", err)
	}

	got, found, err := s.LoadSandbox(context.Background(), sandbox.KindAWS)
	if err != nil {
		t.Fatalf("LoadSandbox: %v", err)
	}
	if !found {
		t.Fatal("LoadSandbox returned found=false after Save")
	}
	if got.Credentials.AccessKey != orig.Credentials.AccessKey ||
		got.Credentials.SecretKey != orig.Credentials.SecretKey {
		t.Errorf("credentials mismatch: %+v", got.Credentials)
	}
	if got.Console != orig.Console {
		t.Errorf("console mismatch: got %+v want %+v", got.Console, orig.Console)
	}
	if got.Identity != orig.Identity {
		t.Errorf("identity mismatch: got %+v want %+v", got.Identity, orig.Identity)
	}
	if !got.ExpiresAt.Equal(orig.ExpiresAt) || !got.StartedAt.Equal(orig.StartedAt) {
		t.Errorf("times mismatch: started=%v expires=%v", got.StartedAt, got.ExpiresAt)
	}
}

func TestStore_LoadSandboxMissing(t *testing.T) {
	s := tempStore(t)
	_, found, err := s.LoadSandbox(context.Background(), sandbox.KindAWS)
	if err != nil {
		t.Fatalf("LoadSandbox: %v", err)
	}
	if found {
		t.Error("found should be false when no state file exists")
	}
}

func TestStore_SaveSandboxPreservesTokens(t *testing.T) {
	s := tempStore(t)
	tokens := sampleTokens()
	if err := s.Save(context.Background(), tokens); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := s.SaveSandbox(context.Background(), sampleSandbox()); err != nil {
		t.Fatalf("SaveSandbox: %v", err)
	}

	got, found, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !found || got.AccessToken != tokens.AccessToken {
		t.Errorf("tokens should survive SaveSandbox: got=%+v found=%v", got, found)
	}
}

func TestStore_SaveTokensPreservesSandbox(t *testing.T) {
	s := tempStore(t)
	if err := s.SaveSandbox(context.Background(), sampleSandbox()); err != nil {
		t.Fatalf("SaveSandbox: %v", err)
	}
	if err := s.Save(context.Background(), sampleTokens()); err != nil {
		t.Fatalf("Save tokens: %v", err)
	}

	got, found, err := s.LoadSandbox(context.Background(), sandbox.KindAWS)
	if err != nil {
		t.Fatalf("LoadSandbox: %v", err)
	}
	if !found {
		t.Error("sandbox cache should survive token Save")
	}
	if got.Credentials.AccessKey == "" {
		t.Error("cached sandbox credentials should survive token Save")
	}
}

func TestStore_ClearSandboxesPreservesTokens(t *testing.T) {
	s := tempStore(t)
	tokens := sampleTokens()
	if err := s.Save(context.Background(), tokens); err != nil {
		t.Fatalf("Save tokens: %v", err)
	}
	if err := s.SaveSandbox(context.Background(), sampleSandbox()); err != nil {
		t.Fatalf("SaveSandbox: %v", err)
	}
	if err := s.ClearSandboxes(context.Background()); err != nil {
		t.Fatalf("ClearSandboxes: %v", err)
	}

	// Sandbox gone.
	_, found, err := s.LoadSandbox(context.Background(), sandbox.KindAWS)
	if err != nil {
		t.Fatalf("LoadSandbox after clear: %v", err)
	}
	if found {
		t.Error("sandbox should be cleared")
	}
	// Tokens intact.
	gotTokens, foundTokens, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("Load after clear: %v", err)
	}
	if !foundTokens || gotTokens.AccessToken != tokens.AccessToken {
		t.Error("session tokens should survive ClearSandboxes")
	}
}

func TestStore_ClearSandboxesOnMissingFileIsNoop(t *testing.T) {
	s := tempStore(t)
	if err := s.ClearSandboxes(context.Background()); err != nil {
		t.Errorf("ClearSandboxes on missing file should be a no-op, got: %v", err)
	}
	if _, statErr := os.Stat(s.Path()); !errors.Is(statErr, fs.ErrNotExist) {
		t.Error("ClearSandboxes must not create a file when none exists")
	}
}

func TestStore_LoadSandboxUnparsableIgnoredNotDeleted(t *testing.T) {
	// A corrupt sandbox timestamp should NOT nuke the session tokens
	// in the same file. The bad sandbox entry is ignored on load.
	s := tempStore(t)
	if err := s.Save(context.Background(), sampleTokens()); err != nil {
		t.Fatalf("Save tokens: %v", err)
	}
	bad := `{
  "version": 1,
  "whizlabs": {
    "user_email": "alice@example.com",
    "access_token": "access-abc",
    "refresh_token": "refresh-def",
    "access_token_expires_at": "2026-04-11T20:00:00Z",
    "refresh_token_expires_at": "2026-04-18T20:00:00Z"
  },
  "sandboxes": {
    "aws": {"kind":"aws","slug":"aws-sandbox","credentials":{"access_key":"AK","secret_key":"SK"},"console":{},"identity":{},"started_at":"not-a-time","expires_at":"also-not"}
  }
}`
	if err := os.WriteFile(s.Path(), []byte(bad), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, found, err := s.LoadSandbox(context.Background(), sandbox.KindAWS)
	if err != nil {
		t.Errorf("LoadSandbox should not error on unparsable entry, got: %v", err)
	}
	if found {
		t.Error("unparsable sandbox should be treated as a miss")
	}
	// Tokens must still load — they live in the same file.
	_, foundTokens, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("Load tokens: %v", err)
	}
	if !foundTokens {
		t.Error("session tokens must survive a corrupt sandbox entry")
	}
}

func TestStore_SaveSandboxFilePerms(t *testing.T) {
	s := tempStore(t)
	if err := s.SaveSandbox(context.Background(), sampleSandbox()); err != nil {
		t.Fatalf("SaveSandbox: %v", err)
	}
	info, err := os.Stat(s.Path())
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if m := info.Mode().Perm(); m != 0o600 {
		t.Errorf("file mode: got %v, want 0600", m)
	}
}

func TestResolveStateDir_Override(t *testing.T) {
	got, err := xdgstore.ResolveStateDir("/custom/override")
	if err != nil {
		t.Fatalf("ResolveStateDir: %v", err)
	}
	if got != "/custom/override" {
		t.Errorf("override: got %q, want /custom/override", got)
	}
}

func TestResolveStateDir_XDG(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/xdg/state")
	t.Setenv("HOME", "/home/test")

	got, err := xdgstore.ResolveStateDir("")
	if err != nil {
		t.Fatalf("ResolveStateDir: %v", err)
	}
	want := filepath.Join("/xdg/state", "whzbox")
	if got != want {
		t.Errorf("xdg: got %q, want %q", got, want)
	}
}

func TestResolveStateDir_HomeFallback(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "/home/test")

	got, err := xdgstore.ResolveStateDir("")
	if err != nil {
		t.Fatalf("ResolveStateDir: %v", err)
	}
	want := filepath.Join("/home/test", ".local", "state", "whzbox")
	if got != want {
		t.Errorf("home fallback: got %q, want %q", got, want)
	}
}
