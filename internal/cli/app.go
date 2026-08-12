package cli

import (
	"log/slog"

	"github.com/spf13/viper"

	"github.com/meigma/whzbox/internal/adapters/awsverify"
	"github.com/meigma/whzbox/internal/adapters/gcpconsole"
	"github.com/meigma/whzbox/internal/adapters/huhprompt"
	"github.com/meigma/whzbox/internal/adapters/whizlabs"
	"github.com/meigma/whzbox/internal/adapters/xdgstore"
	"github.com/meigma/whzbox/internal/config"
	"github.com/meigma/whzbox/internal/core/clock"
	"github.com/meigma/whzbox/internal/core/sandbox"
	"github.com/meigma/whzbox/internal/core/session"
	"github.com/meigma/whzbox/internal/logging"
)

// App is the dependency container wired up in PersistentPreRunE and
// shared across all subcommands.
//
// The fields are exported so tests can construct an App directly with
// fake services, bypassing the production adapter wiring.
type App struct {
	Config config.Config
	Logger *slog.Logger
	Clock  clock.Clock
	Build  BuildInfo

	Environ func() []string
	Getenv  func(string) string

	Session *session.Service
	Sandbox *sandbox.Service
}

// newApp loads config from the supplied Viper instance and constructs
// the production dependency graph: xdg file store, whizlabs HTTP
// client, huh interactive prompt, and the session service that ties
// them together.
func newApp(vp *viper.Viper, options Options) (*App, error) {
	cfg, err := config.Load(vp)
	if err != nil {
		return nil, err
	}

	logger := logging.NewWith(cfg, options.Err)
	realClock := clock.Real{}

	stateDir, err := xdgstore.ResolveStateDir(cfg.StateDir)
	if err != nil {
		return nil, err
	}
	store, err := xdgstore.New(stateDir, logger)
	if err != nil {
		return nil, err
	}

	whiz := whizlabs.NewClient(cfg.Whizlabs, logger)
	prompt := huhprompt.New(options.In, options.Err)

	sessionSvc := session.NewService(whiz, store, prompt, realClock, logger)

	awsProv := awsverify.New("us-east-1")
	gcpProv := gcpconsole.New()
	sandboxSvc := sandbox.NewService(
		sessionSvc,
		whiz,
		map[sandbox.Kind]sandbox.Provider{
			sandbox.KindAWS: awsProv,
			sandbox.KindGCP: gcpProv,
		},
		store.SandboxStore(),
		realClock,
		logger,
	)

	return &App{
		Config:  cfg,
		Logger:  logger,
		Clock:   realClock,
		Build:   options.Build,
		Environ: options.Environ,
		Getenv:  options.Getenv,
		Session: sessionSvc,
		Sandbox: sandboxSvc,
	}, nil
}

// newMetadataApp loads only the pieces needed by the version command.
// It deliberately avoids creating state or network adapters.
func newMetadataApp(vp *viper.Viper, options Options) (*App, error) {
	cfg, err := config.Load(vp)
	if err != nil {
		return nil, err
	}
	return &App{
		Config:  cfg,
		Logger:  logging.NewWith(cfg, options.Err),
		Clock:   clock.Real{},
		Build:   options.Build,
		Environ: options.Environ,
		Getenv:  options.Getenv,
	}, nil
}
