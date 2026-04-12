package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/meigma/whzbox/internal/config"
)

func TestResolveLevel(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		want slog.Level
	}{
		{
			name: "default info",
			cfg:  config.Config{LogLevel: "info"},
			want: slog.LevelInfo,
		},
		{
			name: "-q maps to error",
			cfg:  config.Config{Quiet: true},
			want: slog.LevelError,
		},
		{
			name: "-v maps to info",
			cfg:  config.Config{Verbose: 1},
			want: slog.LevelInfo,
		},
		{
			name: "-vv maps to debug",
			cfg:  config.Config{Verbose: 2},
			want: slog.LevelDebug,
		},
		{
			name: "-vvv still debug (clamped)",
			cfg:  config.Config{Verbose: 3},
			want: slog.LevelDebug,
		},
		{
			name: "--log-level debug overrides -q",
			cfg:  config.Config{LogLevel: "debug", Quiet: true},
			want: slog.LevelDebug,
		},
		{
			name: "--log-level warn",
			cfg:  config.Config{LogLevel: "warn"},
			want: slog.LevelWarn,
		},
		{
			name: "--log-level error",
			cfg:  config.Config{LogLevel: "error"},
			want: slog.LevelError,
		},
		{
			name: "-v beats -q",
			cfg:  config.Config{Verbose: 1, Quiet: true},
			want: slog.LevelInfo,
		},
		{
			name: "-vv beats -q",
			cfg:  config.Config{Verbose: 2, Quiet: true},
			want: slog.LevelDebug,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveLevel(tt.cfg)
			if got != tt.want {
				t.Errorf("resolveLevel(%+v) = %v, want %v", tt.cfg, got, tt.want)
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		in   string
		want slog.Level
		ok   bool
	}{
		{"debug", slog.LevelDebug, true},
		{"DEBUG", slog.LevelDebug, true},
		{"info", slog.LevelInfo, true},
		{"warn", slog.LevelWarn, true},
		{"warning", slog.LevelWarn, true},
		{"error", slog.LevelError, true},
		{"loud", 0, false},
		{"", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := parseLevel(tt.in)
			if ok != tt.ok {
				t.Errorf("parseLevel(%q) ok = %v, want %v", tt.in, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestNewWith_DebugEmitsDebugLines(t *testing.T) {
	var buf bytes.Buffer
	cfg := config.Config{LogLevel: "debug"}
	l := NewWith(cfg, &buf)

	l.Debug("peek-a-boo")

	if !strings.Contains(buf.String(), "peek-a-boo") {
		t.Errorf("expected debug line in output, got: %q", buf.String())
	}
}

func TestNewWith_InfoSuppressesDebug(t *testing.T) {
	var buf bytes.Buffer
	cfg := config.Config{LogLevel: "info"}
	l := NewWith(cfg, &buf)

	l.Debug("do-not-show")
	l.Info("do-show")

	out := buf.String()
	if strings.Contains(out, "do-not-show") {
		t.Errorf("debug line leaked at info level: %q", out)
	}
	if !strings.Contains(out, "do-show") {
		t.Errorf("info line missing: %q", out)
	}
}

func TestNewWith_QuietSuppressesInfo(t *testing.T) {
	var buf bytes.Buffer
	cfg := config.Config{Quiet: true}
	l := NewWith(cfg, &buf)

	l.Info("silenced")
	l.Error("screamed")

	out := buf.String()
	if strings.Contains(out, "silenced") {
		t.Errorf("info line leaked in quiet mode: %q", out)
	}
	if !strings.Contains(out, "screamed") {
		t.Errorf("error line missing in quiet mode: %q", out)
	}
}

func TestNewWith_NoColor(t *testing.T) {
	// Ensure no leftover NO_COLOR from other tests.
	t.Setenv("NO_COLOR", "")

	var buf bytes.Buffer
	cfg := config.Config{LogLevel: "info", NoColor: true}
	l := NewWith(cfg, &buf)

	l.Info("plaintext")

	out := buf.String()
	if strings.Contains(out, "\x1b[") {
		t.Errorf("expected no ANSI escape codes, got: %q", out)
	}
	if !strings.Contains(out, "plaintext") {
		t.Errorf("expected message in output, got: %q", out)
	}
}
