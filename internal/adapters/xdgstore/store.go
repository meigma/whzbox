package xdgstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/meigma/whzbox/internal/core/session"
)

const (
	stateFileName = "state.json"
	schemaVersion = 1
)

// File and directory permissions. The state file contains bearer tokens,
// so both must be owner-only. Any deviation causes Load to refuse to
// read the file rather than silently accepting wider permissions.
const (
	dirMode  fs.FileMode = 0o700
	fileMode fs.FileMode = 0o600
)

// Store implements session.TokenStore backed by a single JSON file at
// $XDG_STATE_HOME/whzbox/state.json (or the equivalent fallback path).
//
// Concurrent use is safe at the process level because Save writes
// atomically (temp file + rename), but cross-process locking is NOT
// provided; two CLI invocations racing on the same state file can
// both succeed with the last writer winning.
type Store struct {
	dir    string
	path   string
	logger *slog.Logger
}

// New returns a Store at the given directory. The directory is created
// with 0700 permissions if it does not yet exist, and existing
// directories are chmod'd to 0700 to protect against wider defaults.
//
// A nil logger is replaced with a discard handler.
func New(stateDir string, logger *slog.Logger) (*Store, error) {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	if stateDir == "" {
		return nil, errors.New("xdgstore: empty state directory")
	}
	if err := os.MkdirAll(stateDir, dirMode); err != nil {
		return nil, fmt.Errorf("create state dir: %w", err)
	}
	if err := os.Chmod(stateDir, dirMode); err != nil {
		return nil, fmt.Errorf("chmod state dir: %w", err)
	}
	return &Store{
		dir:    stateDir,
		path:   filepath.Join(stateDir, stateFileName),
		logger: logger,
	}, nil
}

// Path returns the full path to the state file.
func (s *Store) Path() string { return s.path }

// ResolveStateDir implements the XDG state-directory lookup used by
// the whzbox CLI:
//
//  1. override (typically from --state-dir or WHZBOX_STATE_DIR)
//  2. $XDG_STATE_HOME/whzbox
//  3. $HOME/.local/state/whzbox
//
// Callers pass an empty override when they want the default resolution.
func ResolveStateDir(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "whzbox"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".local", "state", "whzbox"), nil
}

// fileFormat is the on-disk representation of the state file. The
// version field lets us evolve the schema without breaking existing
// installs — an unknown version is treated as corrupt state.
type fileFormat struct {
	Version  int       `json:"version"`
	Whizlabs tokensDTO `json:"whizlabs"`
}

type tokensDTO struct {
	UserEmail             string `json:"user_email"`
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	AccessTokenExpiresAt  string `json:"access_token_expires_at"`
	RefreshTokenExpiresAt string `json:"refresh_token_expires_at"`
}

// Load implements session.TokenStore. A missing file is NOT an error —
// callers see (zero tokens, false, nil). Corrupt or wrong-schema files
// are self-healed (deleted) with a warning log, which is safer than
// surfacing a hard error to the CLI on every invocation.
func (s *Store) Load(ctx context.Context) (session.Tokens, bool, error) {
	info, err := os.Stat(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return session.Tokens{}, false, nil
	}
	if err != nil {
		return session.Tokens{}, false, fmt.Errorf("stat state file: %w", err)
	}

	// Refuse to load a file with permissions wider than 0600. This is
	// a defence-in-depth check; a tampered file with secrets in it is
	// safer to ignore than to use.
	if m := info.Mode().Perm(); m != fileMode {
		return session.Tokens{}, false, fmt.Errorf("state file %s has mode %v, want %v", s.path, m, fileMode)
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		return session.Tokens{}, false, fmt.Errorf("read state file: %w", err)
	}

	var f fileFormat
	err = json.Unmarshal(data, &f)
	if err != nil {
		s.logger.WarnContext(ctx, "state file is corrupt, removing", "path", s.path, "err", err)
		_ = os.Remove(s.path)
		return session.Tokens{}, false, nil
	}
	if f.Version != schemaVersion {
		s.logger.WarnContext(ctx, "state file has unknown schema version, removing",
			"path", s.path, "version", f.Version)
		_ = os.Remove(s.path)
		return session.Tokens{}, false, nil
	}

	t, err := tokensFromDTO(f.Whizlabs)
	if err != nil {
		s.logger.WarnContext(ctx, "state file has unparsable token data, removing", "path", s.path, "err", err)
		_ = os.Remove(s.path)
		return session.Tokens{}, false, nil
	}
	return t, true, nil
}

// Save implements session.TokenStore. It writes atomically: marshal
// -> write temp file with 0600 -> chmod -> rename. If any step fails
// the original file is untouched.
func (s *Store) Save(_ context.Context, t session.Tokens) error {
	f := fileFormat{
		Version:  schemaVersion,
		Whizlabs: dtoFromTokens(t),
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	tmp := s.path + ".tmp"
	err = os.WriteFile(tmp, data, fileMode)
	if err != nil {
		return fmt.Errorf("write temp state file: %w", err)
	}
	// Belt-and-suspenders: os.WriteFile honours the umask, so ensure
	// the final mode is exactly 0600 regardless of the environment.
	err = os.Chmod(tmp, fileMode)
	if err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("chmod temp state file: %w", err)
	}
	err = os.Rename(tmp, s.path)
	if err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename temp state file: %w", err)
	}
	return nil
}

// Clear implements session.TokenStore. A missing file is not an error.
func (s *Store) Clear(_ context.Context) error {
	err := os.Remove(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("remove state file: %w", err)
	}
	return nil
}

// dtoFromTokens converts a domain Tokens into the on-disk shape. Times
// are serialised as RFC 3339 strings.
func dtoFromTokens(t session.Tokens) tokensDTO {
	return tokensDTO{
		UserEmail:             t.UserEmail,
		AccessToken:           t.AccessToken,
		RefreshToken:          t.RefreshToken,
		AccessTokenExpiresAt:  t.AccessTokenExpiresAt.Format(time.RFC3339),
		RefreshTokenExpiresAt: t.RefreshTokenExpiresAt.Format(time.RFC3339),
	}
}

// tokensFromDTO converts an on-disk DTO back into a domain Tokens.
// A parse error here is bubbled up so Load can self-heal.
func tokensFromDTO(d tokensDTO) (session.Tokens, error) {
	accessExp, err := time.Parse(time.RFC3339, d.AccessTokenExpiresAt)
	if err != nil {
		return session.Tokens{}, fmt.Errorf("parse access_token_expires_at: %w", err)
	}
	refreshExp, err := time.Parse(time.RFC3339, d.RefreshTokenExpiresAt)
	if err != nil {
		return session.Tokens{}, fmt.Errorf("parse refresh_token_expires_at: %w", err)
	}
	return session.Tokens{
		UserEmail:             d.UserEmail,
		AccessToken:           d.AccessToken,
		RefreshToken:          d.RefreshToken,
		AccessTokenExpiresAt:  accessExp,
		RefreshTokenExpiresAt: refreshExp,
	}, nil
}
