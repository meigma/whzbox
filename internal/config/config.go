package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// ErrInvalidLogLevel is returned when --log-level or WHZBOX_LOG_LEVEL is not
// one of the recognised values.
var ErrInvalidLogLevel = errors.New("invalid log level")

// Config is the fully resolved application configuration, populated from
// (in order of precedence) flags, environment variables, and built-in defaults.
type Config struct {
	// LogLevel is the absolute log level requested via --log-level or
	// WHZBOX_LOG_LEVEL. Must be one of: debug, info, warn, error.
	LogLevel string `mapstructure:"log-level"`

	// Verbose is the -v count. Higher values mean more output.
	Verbose int `mapstructure:"verbose"`

	// Quiet suppresses all output below ERROR level.
	Quiet bool `mapstructure:"quiet"`

	// NoColor disables ANSI colour output regardless of TTY detection.
	NoColor bool `mapstructure:"no-color"`

	// AssumeYes skips interactive confirmation prompts.
	AssumeYes bool `mapstructure:"yes"`

	// StateDir overrides the directory where the session state file lives.
	// Empty means "use the XDG default".
	StateDir string `mapstructure:"state-dir"`

	// Whizlabs holds the API endpoints. Defaults point at production.
	Whizlabs WhizlabsConfig `mapstructure:"whizlabs"`
}

// WhizlabsConfig holds API endpoint overrides. Only useful for tests and
// local development against a fake server.
type WhizlabsConfig struct {
	BaseURL string `mapstructure:"base-url"`
	PlayURL string `mapstructure:"play-url"`
}

// Default endpoints used by Load when neither flag nor env var sets them.
const (
	DefaultWhizlabsBaseURL = "https://fq6dv85p2h.execute-api.us-east-1.amazonaws.com"
	DefaultWhizlabsPlayURL = "https://play.whizlabs.com"
)

// envKeys are all config keys that should be resolvable from environment
// variables. Viper's AutomaticEnv does not walk Unmarshal targets, so keys
// must be registered explicitly via BindEnv or SetDefault for Get calls
// inside Unmarshal to see them.
var envKeys = []string{ //nolint:gochecknoglobals // static config key registry
	"log-level",
	"verbose",
	"quiet",
	"no-color",
	"yes",
	"state-dir",
	"whizlabs.base-url",
	"whizlabs.play-url",
}

// Load reads configuration from the supplied Viper instance and returns a
// validated Config. Callers are expected to have already configured Viper
// with an env prefix, key replacer, and any flag bindings.
func Load(vp *viper.Viper) (Config, error) {
	// Built-in defaults. Note: log-level has no default on purpose — an
	// empty LogLevel means "fall through to -v/-q/info", which lets the
	// verbose and quiet flags work without being masked.
	vp.SetDefault("whizlabs.base-url", DefaultWhizlabsBaseURL)
	vp.SetDefault("whizlabs.play-url", DefaultWhizlabsPlayURL)

	// Explicit env bindings so every key is resolvable during Unmarshal.
	for _, k := range envKeys {
		if err := vp.BindEnv(k); err != nil {
			return Config{}, fmt.Errorf("bind env %q: %w", k, err)
		}
	}

	var cfg Config
	if err := vp.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate reports a typed error if any field has an unacceptable value.
//
// An empty LogLevel is valid and means "derive the level from -v/-q, or
// fall back to info". Only explicitly set values are rejected when unknown.
func (c Config) Validate() error {
	switch strings.ToLower(c.LogLevel) {
	case "", "debug", "info", "warn", "warning", "error":
		return nil
	default:
		return fmt.Errorf("%w: %q (want one of: debug, info, warn, error)", ErrInvalidLogLevel, c.LogLevel)
	}
}
