package gcpconsole_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meigma/whzbox/internal/adapters/gcpconsole"
	"github.com/meigma/whzbox/internal/core/sandbox"
)

func TestProviderReportsConsoleOnlyContract(t *testing.T) {
	provider := gcpconsole.New()

	identity, err := provider.VerifyCredentials(context.Background(), sandbox.Credentials{})

	assert.Equal(t, sandbox.KindGCP, provider.Kind())
	assert.Equal(t, "gcp-sandbox", provider.Slug())
	assert.Empty(t, identity)
	require.ErrorIs(t, err, sandbox.ErrVerificationUnsupported)
	assert.Empty(t, provider.Env(nil))
}
