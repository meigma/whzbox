package cli

import (
	"io"
	"os"

	"github.com/spf13/viper"
)

// Options supplies process-level dependencies to the command tree.
type Options struct {
	In   io.Reader
	Out  io.Writer
	Err  io.Writer
	Args []string

	Viper   *viper.Viper
	Build   BuildInfo
	Environ func() []string
	Getenv  func(string) string
}

// DefaultOptions returns the production command dependencies.
func DefaultOptions() Options {
	return Options{
		In:      os.Stdin,
		Out:     os.Stdout,
		Err:     os.Stderr,
		Viper:   viper.New(),
		Build:   CurrentBuildInfo(),
		Environ: os.Environ,
		Getenv:  os.Getenv,
	}
}

func (o Options) withDefaults() Options {
	defaults := DefaultOptions()
	if o.In == nil {
		o.In = defaults.In
	}
	if o.Out == nil {
		o.Out = defaults.Out
	}
	if o.Err == nil {
		o.Err = defaults.Err
	}
	if o.Viper == nil {
		o.Viper = defaults.Viper
	}
	if o.Build == (BuildInfo{}) {
		o.Build = defaults.Build
	}
	if o.Environ == nil {
		o.Environ = defaults.Environ
	}
	if o.Getenv == nil {
		o.Getenv = defaults.Getenv
	}
	return o
}
