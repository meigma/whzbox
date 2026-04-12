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
