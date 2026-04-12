package config_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/meigma/whzbox/internal/config"
)

func newViper(t *testing.T) *viper.Viper {
	t.Helper()
	vp := viper.New()
	vp.SetEnvPrefix("WHZBOX")
	vp.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	vp.AutomaticEnv()
	return vp
}

func TestLoad_Defaults(t *testing.T) {
	cfg, err := config.Load(newViper(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// LogLevel is deliberately empty by default; resolution happens in
	// the logging package based on -v / -q precedence.
	if cfg.LogLevel != "" {
		t.Errorf("LogLevel: got %q, want empty (unset)", cfg.LogLevel)
	}
	if cfg.Verbose != 0 {
		t.Errorf("Verbose: got %d, want 0", cfg.Verbose)
	}
	if cfg.Quiet {
		t.Error("Quiet: got true, want false")
	}
	if cfg.NoColor {
		t.Error("NoColor: got true, want false")
	}
	if cfg.AssumeYes {
		t.Error("AssumeYes: got true, want false")
	}
	if cfg.Whizlabs.BaseURL != config.DefaultWhizlabsBaseURL {
		t.Errorf("Whizlabs.BaseURL: got %q, want %q", cfg.Whizlabs.BaseURL, config.DefaultWhizlabsBaseURL)
	}
	if cfg.Whizlabs.PlayURL != config.DefaultWhizlabsPlayURL {
		t.Errorf("Whizlabs.PlayURL: got %q, want %q", cfg.Whizlabs.PlayURL, config.DefaultWhizlabsPlayURL)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("WHZBOX_LOG_LEVEL", "debug")
	cfg, err := config.Load(newViper(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel: got %q, want %q", cfg.LogLevel, "debug")
	}
}

func TestLoad_FlagOverridesEnv(t *testing.T) {
	t.Setenv("WHZBOX_LOG_LEVEL", "debug")

	vp := newViper(t)
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.String("log-level", "info", "")
	if err := fs.Set("log-level", "warn"); err != nil {
		t.Fatalf("fs.Set: %v", err)
	}
	if err := vp.BindPFlags(fs); err != nil {
		t.Fatalf("BindPFlags: %v", err)
	}

	cfg, err := config.Load(vp)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("LogLevel: got %q, want %q", cfg.LogLevel, "warn")
	}
}

func TestLoad_EnvVerbose(t *testing.T) {
	t.Setenv("WHZBOX_VERBOSE", "2")
	cfg, err := config.Load(newViper(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Verbose != 2 {
		t.Errorf("Verbose: got %d, want 2", cfg.Verbose)
	}
}

func TestLoad_EnvNoColor(t *testing.T) {
	t.Setenv("WHZBOX_NO_COLOR", "true")
	cfg, err := config.Load(newViper(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.NoColor {
		t.Error("NoColor: got false, want true")
	}
}

func TestLoad_InvalidLevel(t *testing.T) {
	t.Setenv("WHZBOX_LOG_LEVEL", "yelling")
	_, err := config.Load(newViper(t))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, config.ErrInvalidLogLevel) {
		t.Errorf("error type: got %v, want ErrInvalidLogLevel", err)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		wantErr bool
	}{
		{"debug", "debug", false},
		{"info", "info", false},
		{"warn", "warn", false},
		{"warning alias", "warning", false},
		{"error", "error", false},
		{"uppercase", "DEBUG", false},
		{"empty (means unset)", "", false},
		{"garbage", "loud", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{LogLevel: tt.level}
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
