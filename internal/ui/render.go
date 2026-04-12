package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/meigma/whzbox/internal/core/sandbox"
	"github.com/meigma/whzbox/internal/core/session"
)

const labelWidth = 22

// fpf writes a formatted line to w, swallowing any write error — the
// rendering layer cannot do anything useful with a failure to write
// to stdout, and errcheck would otherwise force every call site to
// explicitly ignore the return value.
func fpf(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

// RenderSandbox writes the credential box for a provisioned sandbox
// to w. The box has a rounded border and contains account info,
// console login, programmatic credentials, and the expiry window.
//
// A nil sandbox is treated as a no-op so callers can unconditionally
// render after Create without nil-checking.
func RenderSandbox(w io.Writer, sb *sandbox.Sandbox) {
	if sb == nil {
		return
	}

	label := lipgloss.NewStyle().Foreground(Dim).Width(labelWidth)

	rows := []struct{ k, v string }{
		{"Account", sb.Identity.Account},
		{"User", sb.Identity.UserID},
		{"ARN", sb.Identity.ARN},
		{"Region", sb.Identity.Region},
		{"", ""},
		{"Expires", formatExpiry(sb.ExpiresAt)},
		{"", ""},
		{"Console", sb.Console.URL},
		{"Username", sb.Console.Username},
		{"Password", sb.Console.Password},
		{"", ""},
		{"AWS_ACCESS_KEY_ID", sb.Credentials.AccessKey},
		{"AWS_SECRET_ACCESS_KEY", sb.Credentials.SecretKey},
	}

	var buf strings.Builder
	for _, r := range rows {
		if r.k == "" && r.v == "" {
			buf.WriteString("\n")
			continue
		}
		buf.WriteString(label.Render(r.k) + "  " + r.v + "\n")
	}
	body := strings.TrimRight(buf.String(), "\n")

	fpf(w, "%s\n\n", SandboxFrame().Render(body))
	fpf(w, "Destroy with:  whzbox destroy\n")
}

// SandboxJSON is the machine-readable shape emitted by --json. It lives
// in the ui package so that internal/core/sandbox stays free of
// serialisation concerns.
type SandboxJSON struct {
	Kind        string       `json:"kind"`
	Slug        string       `json:"slug"`
	Credentials CredsJSON    `json:"credentials"`
	Console     ConsoleJSON  `json:"console"`
	Identity    IdentityJSON `json:"identity"`
	StartedAt   time.Time    `json:"started_at"`
	ExpiresAt   time.Time    `json:"expires_at"`
}

type CredsJSON struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

type ConsoleJSON struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type IdentityJSON struct {
	Account string `json:"account"`
	UserID  string `json:"user_id"`
	ARN     string `json:"arn"`
	Region  string `json:"region"`
}

// sandboxToJSON converts the domain Sandbox into its JSON DTO.
func sandboxToJSON(sb *sandbox.Sandbox) SandboxJSON {
	return SandboxJSON{
		Kind: string(sb.Kind),
		Slug: sb.Slug,
		Credentials: CredsJSON{
			AccessKey: sb.Credentials.AccessKey,
			SecretKey: sb.Credentials.SecretKey,
		},
		Console: ConsoleJSON{
			URL:      sb.Console.URL,
			Username: sb.Console.Username,
			Password: sb.Console.Password,
		},
		Identity: IdentityJSON{
			Account: sb.Identity.Account,
			UserID:  sb.Identity.UserID,
			ARN:     sb.Identity.ARN,
			Region:  sb.Identity.Region,
		},
		StartedAt: sb.StartedAt,
		ExpiresAt: sb.ExpiresAt,
	}
}

// RenderSandboxJSON writes a single sandbox as indented JSON.
func RenderSandboxJSON(w io.Writer, sb *sandbox.Sandbox) error {
	if sb == nil {
		_, err := fmt.Fprintln(w, "null")
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(sandboxToJSON(sb))
}

// RenderSandboxList writes a compact table of cached sandboxes: one
// row per entry, with a status column derived from the supplied "now".
// An empty slice prints a "(no sandboxes cached)" marker.
func RenderSandboxList(w io.Writer, sbs []*sandbox.Sandbox, now time.Time) {
	if len(sbs) == 0 {
		fpf(w, "(no sandboxes cached)\n")
		return
	}
	fpf(w, "%-6s  %-14s  %-7s  %s\n", "KIND", "ACCOUNT", "STATUS", "EXPIRES")
	for _, sb := range sbs {
		if sb == nil {
			continue
		}
		status := "active"
		if !sb.ExpiresAt.After(now) {
			status = "expired"
		}
		fpf(w, "%-6s  %-14s  %-7s  %s\n",
			string(sb.Kind),
			sb.Identity.Account,
			status,
			formatExpiry(sb.ExpiresAt),
		)
	}
}

// RenderSandboxListJSON writes a JSON array of sandboxes. A nil/empty
// slice renders as "[]" so callers can pipe through jq without a null
// special case.
func RenderSandboxListJSON(w io.Writer, sbs []*sandbox.Sandbox) error {
	out := make([]SandboxJSON, 0, len(sbs))
	for _, sb := range sbs {
		if sb == nil {
			continue
		}
		out = append(out, sandboxToJSON(sb))
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// RenderStatus writes the status view (session info)
// to w. When the user has never logged in, the session block shows
// "(not logged in)".
func RenderStatus(w io.Writer, tokens session.Tokens, found bool) {
	fpf(w, "Session\n")
	if !found {
		fpf(w, "  (not logged in)\n")
	} else {
		if tokens.UserEmail != "" {
			fpf(w, "  Email    %s\n", tokens.UserEmail)
		}
		fpf(w, "  Expires  %s\n", formatExpiry(tokens.AccessTokenExpiresAt))
		if tokens.RefreshToken != "" {
			fpf(w, "  Refresh  %s\n", formatExpiry(tokens.RefreshTokenExpiresAt))
		}
	}
}

// formatExpiry renders a timestamp as "2026-04-11 20:00:00 UTC (in 14h 32m)".
// Zero times become "unknown". Past times become "expired".
func formatExpiry(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	abs := t.UTC().Format("2006-01-02 15:04:05 UTC")
	d := time.Until(t).Round(time.Second)
	if d <= 0 {
		return abs + " (expired)"
	}
	return fmt.Sprintf("%s (in %s)", abs, humanDuration(d))
}

// humanDuration formats a positive duration as "Xh Ym" (or finer when
// under an hour). Chosen for legibility, not precision.
func humanDuration(d time.Duration) string {
	h := int(d / time.Hour)
	m := int((d % time.Hour) / time.Minute)
	s := int((d % time.Minute) / time.Second)
	switch {
	case h > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	case m > 0:
		return fmt.Sprintf("%dm %ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}
