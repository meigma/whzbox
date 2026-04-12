package ui

import (
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

	var body string
	var bodySb52 strings.Builder
	for _, r := range rows {
		if r.k == "" && r.v == "" {
			bodySb52.WriteString("\n")
			continue
		}
		bodySb52.WriteString(label.Render(r.k) + "  " + r.v + "\n")
	}
	body += bodySb52.String()
	// Trim the final newline so the frame fits snugly.
	if len(body) > 0 && body[len(body)-1] == '\n' {
		body = body[:len(body)-1]
	}

	fpf(w, "%s\n\n", SandboxFrame().Render(body))
	fpf(w, "Destroy with:  whzbox destroy\n")
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
