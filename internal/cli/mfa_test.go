package cli

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meigma/whzbox/internal/core/clock"
	"github.com/meigma/whzbox/internal/core/sandbox"
)

type mfaStore struct {
	sandbox *sandbox.Sandbox
}

func (s mfaStore) Load(_ context.Context, kind sandbox.Kind) (*sandbox.Sandbox, bool, error) {
	return s.sandbox, s.sandbox != nil && s.sandbox.Kind == kind, nil
}

func (s mfaStore) LoadAll(context.Context) ([]*sandbox.Sandbox, error) { return nil, nil }
func (s mfaStore) Save(context.Context, *sandbox.Sandbox) error        { return nil }
func (s mfaStore) ClearAll(context.Context) error                      { return nil }

func TestMFACommand_PrintsFreshAzureCode(t *testing.T) {
	now := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)
	mgr := &stubManager{mfaResult: "123456"}
	provider := &stubVerifier{kind: sandbox.KindAzure, slug: "azure-sandbox"}
	service := sandbox.NewService(
		&stubAuth{},
		mgr,
		testProviderMap(provider),
		mfaStore{sandbox: &sandbox.Sandbox{
			Kind:      sandbox.KindAzure,
			Slug:      "azure-sandbox",
			ExpiresAt: now.Add(time.Hour),
		}},
		&clock.Fake{T: now},
		nil,
	)
	app := &App{Sandbox: service}

	cmd := newMFACommand(&app)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"azure"})

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "123456\n", out.String())
}

func TestMFACommand_RejectsUnsupportedProvider(t *testing.T) {
	app := &App{}
	cmd := newMFACommand(&app)
	cmd.SetArgs([]string{"aws"})
	require.Error(t, cmd.Execute())
}
