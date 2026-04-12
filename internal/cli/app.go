package cli

import (
	"log/slog"

	"github.com/spf13/viper"

	"github.com/meigma/whzbox/internal/adapters/awsverify"
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
// fake services, bypassing NewApp's real-adapter wiring.
type App struct {
	Config config.Config
	Logger *slog.Logger
	Clock  clock.Clock

	Session *session.Service
	Sandbox *sandbox.Service
}

// NewApp loads config from the supplied Viper instance and constructs
// the production dependency graph: xdg file store, whizlabs HTTP
// client, huh interactive prompt, and the session service that ties
// them together.
func NewApp(vp *viper.Viper) (*App, error) {
	cfg, err := config.Load(vp)
	if err != nil {
		return nil, err
	}

	logger := logging.New(cfg)
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
	prompt := huhprompt.New()

	sessionSvc := session.NewService(whiz, store, prompt, realClock, logger)

	awsProv := awsverify.New("us-east-1")
	sandboxSvc := sandbox.NewService(
		sessionSvc,
		whiz,
		map[sandbox.Kind]sandbox.Provider{
			sandbox.KindAWS: awsProv,
		},
		store.SandboxStore(),
		realClock,
		logger,
	)

	return &App{
		Config:  cfg,
		Logger:  logger,
		Clock:   realClock,
		Session: sessionSvc,
		Sandbox: sandboxSvc,
	}, nil
}

// NewMetadataApp loads only the pieces needed by the version command.
// It deliberately avoids creating state or network adapters.
func NewMetadataApp(vp *viper.Viper) (*App, error) {
	cfg, err := config.Load(vp)
	if err != nil {
		return nil, err
	}
	return &App{
		Config: cfg,
		Logger: logging.New(cfg),
		Clock:  clock.Real{},
	}, nil
}
