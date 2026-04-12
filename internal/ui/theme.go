package ui

import (
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	charmlog "charm.land/log/v2"
)

// Shared palette. Every form, frame, and log level pulls from these so the
// CLI has one visual identity.
var (
	Accent  = lipgloss.Color("#874BFD") //nolint:gochecknoglobals // shared UI palette
	Success = lipgloss.Color("#02BA84") //nolint:gochecknoglobals // shared UI palette
	Warn    = lipgloss.Color("#F2B033") //nolint:gochecknoglobals // shared UI palette
	Danger  = lipgloss.Color("#EF4444") //nolint:gochecknoglobals // shared UI palette
	Dim     = lipgloss.Color("245")     //nolint:gochecknoglobals // shared UI palette
)

// HuhTheme builds a huh styles set from the shared palette. isDark is passed
// through to huh.ThemeBase for baseline contrast handling.
func HuhTheme(isDark bool) *huh.Styles {
	t := huh.ThemeBase(isDark)
	t.Focused.Title = t.Focused.Title.Foreground(Accent).Bold(true)
	t.Focused.FocusedButton = t.Focused.FocusedButton.
		Foreground(lipgloss.Color("#FFFDF5")).
		Background(Accent)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(Success)
	t.Group.Title = t.Focused.Title
	return t
}

// LogStyles returns a charm log styles set themed to the shared palette.
func LogStyles() *charmlog.Styles {
	s := charmlog.DefaultStyles()
	s.Levels[charmlog.DebugLevel] = lipgloss.NewStyle().SetString("DEBUG").Foreground(Dim)
	s.Levels[charmlog.InfoLevel] = lipgloss.NewStyle().SetString("INFO").Foreground(Accent)
	s.Levels[charmlog.WarnLevel] = lipgloss.NewStyle().SetString("WARN").Foreground(Warn)
	s.Levels[charmlog.ErrorLevel] = lipgloss.NewStyle().SetString("ERROR").Foreground(Danger)
	return s
}

// SandboxFrame returns the lipgloss style used for the credential box rendered
// after a successful `create`.
func SandboxFrame() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Success).
		Padding(1, 2) //nolint:mnd // visual padding
}
