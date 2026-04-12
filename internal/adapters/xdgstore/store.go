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

	"github.com/meigma/whzbox/internal/core/sandbox"
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
	Version   int                   `json:"version"`
	Whizlabs  tokensDTO             `json:"whizlabs"`
	Sandboxes map[string]sandboxDTO `json:"sandboxes,omitempty"`
}

type tokensDTO struct {
	UserEmail             string `json:"user_email"`
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	AccessTokenExpiresAt  string `json:"access_token_expires_at"`
	RefreshTokenExpiresAt string `json:"refresh_token_expires_at"`
}

type sandboxDTO struct {
	Kind      string         `json:"kind"`
	Slug      string         `json:"slug"`
	Verified  *bool          `json:"verified,omitempty"`
	Creds     credentialsDTO `json:"credentials"`
	Console   consoleDTO     `json:"console"`
	Identity  identityDTO    `json:"identity"`
	StartedAt string         `json:"started_at"`
	ExpiresAt string         `json:"expires_at"`
}

type credentialsDTO struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

type consoleDTO struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type identityDTO struct {
	Account string `json:"account"`
	UserID  string `json:"user_id"`
	ARN     string `json:"arn"`
	Region  string `json:"region"`
}

// Load implements session.TokenStore. A missing file is NOT an error —
// callers see (zero tokens, false, nil). A state file with an empty
// auth section but cached sandboxes is treated as "logged out", not
// corrupt. Corrupt or wrong-schema files are self-healed (deleted) with
// a warning log, which is safer than surfacing a hard error to the CLI
// on every invocation.
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
	if tokensDTOEmpty(f.Whizlabs) {
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
//
// Save preserves any cached sandboxes already on disk (read-modify-
// write) so that a session refresh does not clobber the sandbox
// cache.
func (s *Store) Save(_ context.Context, t session.Tokens) error {
	f := s.readCurrent()
	f.Whizlabs = dtoFromTokens(t)
	return s.writeAtomic(f)
}

// readCurrent reads the current on-disk state, returning a zero
// fileFormat (at the current schema version) if the file is missing,
// unreadable, corrupt, or at an unknown version. Used by writers to
// merge their update into whatever is already stored without blowing
// away unrelated fields.
func (s *Store) readCurrent() fileFormat {
	zero := fileFormat{Version: schemaVersion}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return zero
	}
	var f fileFormat
	if uerr := json.Unmarshal(data, &f); uerr != nil || f.Version != schemaVersion {
		return zero
	}
	return f
}

// writeAtomic marshals f and writes it to the state path via
// temp-file + rename, ensuring the final mode is exactly 0600.
func (s *Store) writeAtomic(f fileFormat) error {
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
// When sandbox cache entries exist in the shared state file, only the
// auth section is cleared so cached sandbox credentials survive logout.
func (s *Store) Clear(_ context.Context) error {
	if _, err := os.Stat(s.path); errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	f := s.readCurrent()
	if len(f.Sandboxes) == 0 {
		err := os.Remove(s.path)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("remove state file: %w", err)
		}
		return nil
	}

	f.Whizlabs = tokensDTO{}
	if stateEmpty(f) {
		err := os.Remove(s.path)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("remove state file: %w", err)
		}
		return nil
	}
	return s.writeAtomic(f)
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

// LoadSandbox returns the cached sandbox for the given kind, or
// (nil, false, nil) when nothing is cached. An unparsable cached
// entry is ignored (treated as a miss) with a warning — the session
// tokens in the same file are preserved, because losing your login
// just to recover from a bad cache entry is a bad trade.
func (s *Store) LoadSandbox(ctx context.Context, kind sandbox.Kind) (*sandbox.Sandbox, bool, error) {
	info, err := os.Stat(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("stat state file: %w", err)
	}
	if m := info.Mode().Perm(); m != fileMode {
		return nil, false, fmt.Errorf("state file %s has mode %v, want %v", s.path, m, fileMode)
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, false, fmt.Errorf("read state file: %w", err)
	}
	var f fileFormat
	if uerr := json.Unmarshal(data, &f); uerr != nil {
		// Whole-file corruption is self-healed by the session token
		// Load path; here we just treat the cache as a miss so the
		// caller proceeds to provision a fresh sandbox.
		s.logger.WarnContext(ctx, "state file is corrupt, ignoring cached sandbox",
			"path", s.path, "err", uerr)
		return nil, false, nil
	}
	if f.Version != schemaVersion {
		return nil, false, nil
	}

	dto, ok := f.Sandboxes[string(kind)]
	if !ok {
		return nil, false, nil
	}
	sb, err := sandboxFromDTO(dto)
	if err != nil {
		s.logger.WarnContext(ctx, "cached sandbox is unparsable, ignoring",
			"path", s.path, "kind", kind, "err", err)
		return nil, false, nil
	}
	return sb, true, nil
}

// LoadAllSandboxes returns every cached sandbox, in unspecified order.
// A missing state file yields an empty slice. Unparsable individual
// entries are skipped with a warning (mirroring LoadSandbox's
// "ignore and proceed" policy), not surfaced as a hard error.
func (s *Store) LoadAllSandboxes(ctx context.Context) ([]*sandbox.Sandbox, error) {
	info, err := os.Stat(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("stat state file: %w", err)
	}
	if m := info.Mode().Perm(); m != fileMode {
		return nil, fmt.Errorf("state file %s has mode %v, want %v", s.path, m, fileMode)
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("read state file: %w", err)
	}
	var f fileFormat
	if uerr := json.Unmarshal(data, &f); uerr != nil {
		s.logger.WarnContext(ctx, "state file is corrupt, ignoring cached sandboxes",
			"path", s.path, "err", uerr)
		return nil, nil
	}
	if f.Version != schemaVersion {
		return nil, nil
	}

	out := make([]*sandbox.Sandbox, 0, len(f.Sandboxes))
	for kind, dto := range f.Sandboxes {
		sb, perr := sandboxFromDTO(dto)
		if perr != nil {
			s.logger.WarnContext(ctx, "cached sandbox is unparsable, ignoring",
				"path", s.path, "kind", kind, "err", perr)
			continue
		}
		out = append(out, sb)
	}
	return out, nil
}

// SaveSandbox writes sb into the cache keyed by its Kind, preserving
// session tokens and any sandbox entries for other kinds.
func (s *Store) SaveSandbox(_ context.Context, sb *sandbox.Sandbox) error {
	if sb == nil {
		return errors.New("xdgstore: nil sandbox")
	}
	f := s.readCurrent()
	if f.Sandboxes == nil {
		f.Sandboxes = map[string]sandboxDTO{}
	}
	f.Sandboxes[string(sb.Kind)] = dtoFromSandbox(sb)
	return s.writeAtomic(f)
}

// ClearSandboxes drops every cached sandbox, preserving session
// tokens. A missing state file is a no-op (nothing to clear).
func (s *Store) ClearSandboxes(_ context.Context) error {
	if _, err := os.Stat(s.path); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	f := s.readCurrent()
	if len(f.Sandboxes) == 0 {
		return nil
	}
	f.Sandboxes = nil
	if stateEmpty(f) {
		err := os.Remove(s.path)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("remove state file: %w", err)
		}
		return nil
	}
	return s.writeAtomic(f)
}

// SandboxStore returns a view of this Store that satisfies
// sandbox.Store. A separate wrapper is needed because *Store already
// has Load/Save/Clear methods with session.TokenStore signatures,
// and Go does not allow two methods with the same name but different
// signatures on a single receiver.
func (s *Store) SandboxStore() sandbox.Store {
	return sandboxView{s: s}
}

type sandboxView struct{ s *Store }

func (v sandboxView) Load(ctx context.Context, kind sandbox.Kind) (*sandbox.Sandbox, bool, error) {
	return v.s.LoadSandbox(ctx, kind)
}

func (v sandboxView) LoadAll(ctx context.Context) ([]*sandbox.Sandbox, error) {
	return v.s.LoadAllSandboxes(ctx)
}

func (v sandboxView) Save(ctx context.Context, sb *sandbox.Sandbox) error {
	return v.s.SaveSandbox(ctx, sb)
}

func (v sandboxView) ClearAll(ctx context.Context) error {
	return v.s.ClearSandboxes(ctx)
}

func dtoFromSandbox(sb *sandbox.Sandbox) sandboxDTO {
	verified := sb.Verified
	return sandboxDTO{
		Kind:     string(sb.Kind),
		Slug:     sb.Slug,
		Verified: &verified,
		Creds: credentialsDTO{
			AccessKey: sb.Credentials.AccessKey,
			SecretKey: sb.Credentials.SecretKey,
		},
		Console: consoleDTO{
			URL:      sb.Console.URL,
			Username: sb.Console.Username,
			Password: sb.Console.Password,
		},
		Identity: identityDTO{
			Account: sb.Identity.Account,
			UserID:  sb.Identity.UserID,
			ARN:     sb.Identity.ARN,
			Region:  sb.Identity.Region,
		},
		StartedAt: sb.StartedAt.Format(time.RFC3339),
		ExpiresAt: sb.ExpiresAt.Format(time.RFC3339),
	}
}

func sandboxFromDTO(d sandboxDTO) (*sandbox.Sandbox, error) {
	startedAt, err := time.Parse(time.RFC3339, d.StartedAt)
	if err != nil {
		return nil, fmt.Errorf("parse started_at: %w", err)
	}
	expiresAt, err := time.Parse(time.RFC3339, d.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("parse expires_at: %w", err)
	}
	return &sandbox.Sandbox{
		Kind:     sandbox.Kind(d.Kind),
		Slug:     d.Slug,
		Verified: sandboxVerified(d),
		Credentials: sandbox.Credentials{
			AccessKey: d.Creds.AccessKey,
			SecretKey: d.Creds.SecretKey,
		},
		Console: sandbox.Console{
			URL:      d.Console.URL,
			Username: d.Console.Username,
			Password: d.Console.Password,
		},
		Identity: sandbox.Identity{
			Account: d.Identity.Account,
			UserID:  d.Identity.UserID,
			ARN:     d.Identity.ARN,
			Region:  d.Identity.Region,
		},
		StartedAt: startedAt,
		ExpiresAt: expiresAt,
	}, nil
}

func tokensDTOEmpty(d tokensDTO) bool {
	return d.UserEmail == "" &&
		d.AccessToken == "" &&
		d.RefreshToken == "" &&
		d.AccessTokenExpiresAt == "" &&
		d.RefreshTokenExpiresAt == ""
}

func stateEmpty(f fileFormat) bool {
	return tokensDTOEmpty(f.Whizlabs) && len(f.Sandboxes) == 0
}

func sandboxVerified(d sandboxDTO) bool {
	if d.Verified != nil {
		return *d.Verified
	}
	return d.Identity.Account != "" || d.Identity.UserID != "" || d.Identity.ARN != ""
}
