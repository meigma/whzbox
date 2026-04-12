package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"

	charmlog "charm.land/log/v2"

	"github.com/meigma/whzbox/internal/config"
	"github.com/meigma/whzbox/internal/ui"
)

// New returns a [slog.Logger] writing to [os.Stderr], styled via the shared
// charm log handler and the UI theme.
func New(cfg config.Config) *slog.Logger {
	return NewWith(cfg, os.Stderr)
}

// NewWith is identical to New but lets tests inject a writer.
func NewWith(cfg config.Config, w io.Writer) *slog.Logger {
	level := resolveLevel(cfg)

	handler := charmlog.NewWithOptions(w, charmlog.Options{
		Level:           toCharmLevel(level),
		ReportTimestamp: level <= slog.LevelDebug,
	})
	handler.SetStyles(ui.LogStyles())

	if wantsNoColor(cfg) {
		// charmlog does not expose a public disable-colour method; the
		// cheapest portable workaround is to honour the NO_COLOR env var,
		// which the underlying lipgloss/termenv stack already respects.
		// Setting it only affects this process, which is what we want.
		_ = os.Setenv("NO_COLOR", "1")
	}

	return slog.New(handler)
}

// resolveLevel implements the precedence documented in PLAN.md §11:
//
//	--log-level > -v count > --quiet > default(info)
//
// Any -v (>= 1) beats --quiet. -vv or higher means debug. --log-level,
// when set, takes absolute precedence and its value is trusted because
// config.Load has already validated it.
func resolveLevel(cfg config.Config) slog.Level {
	if cfg.LogLevel != "" {
		if l, ok := parseLevel(cfg.LogLevel); ok {
			return l
		}
	}
	if cfg.Verbose >= 2 { //nolint:mnd // -vv means debug
		return slog.LevelDebug
	}
	if cfg.Verbose >= 1 {
		return slog.LevelInfo
	}
	if cfg.Quiet {
		return slog.LevelError
	}
	return slog.LevelInfo
}

func parseLevel(s string) (slog.Level, bool) {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug, true
	case "info":
		return slog.LevelInfo, true
	case "warn", "warning":
		return slog.LevelWarn, true
	case "error":
		return slog.LevelError, true
	}
	return 0, false
}

func toCharmLevel(l slog.Level) charmlog.Level {
	switch {
	case l <= slog.LevelDebug:
		return charmlog.DebugLevel
	case l <= slog.LevelInfo:
		return charmlog.InfoLevel
	case l <= slog.LevelWarn:
		return charmlog.WarnLevel
	default:
		return charmlog.ErrorLevel
	}
}

func wantsNoColor(cfg config.Config) bool {
	if cfg.NoColor {
		return true
	}
	// Honour the de-facto standard even if the caller didn't pass --no-color.
	return os.Getenv("NO_COLOR") != ""
}
