package azureconsole

import (
	"context"

	"github.com/meigma/whzbox/internal/core/sandbox"
)

// Provider describes Whizlabs' browser-console-only Azure sandbox.
type Provider struct{}

// New returns an Azure console provider.
func New() *Provider { return &Provider{} }

// Kind returns the provider kind.
func (*Provider) Kind() sandbox.Kind { return sandbox.KindAzure }

// Slug returns the Whizlabs task slug.
func (*Provider) Slug() string { return "azure-sandbox" }

// VerifyCredentials reports that Whizlabs does not supply a
// programmatic Azure credential that can be verified independently.
func (*Provider) VerifyCredentials(context.Context, sandbox.Credentials) (sandbox.Identity, error) {
	return sandbox.Identity{}, sandbox.ErrVerificationUnsupported
}

// Env returns no variables because the Azure sandbox is console-only.
func (*Provider) Env(*sandbox.Sandbox) []string { return nil }
