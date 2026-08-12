package azureconsole_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meigma/whzbox/internal/adapters/azureconsole"
	"github.com/meigma/whzbox/internal/core/sandbox"
)

func TestProviderDescribesConsoleOnlyAzure(t *testing.T) {
	provider := azureconsole.New()

	identity, err := provider.VerifyCredentials(context.Background(), sandbox.Credentials{})

	assert.Equal(t, sandbox.KindAzure, provider.Kind())
	assert.Equal(t, "azure-sandbox", provider.Slug())
	require.ErrorIs(t, err, sandbox.ErrVerificationUnsupported)
	assert.Empty(t, identity)
	assert.Empty(t, provider.Env(nil))
}
