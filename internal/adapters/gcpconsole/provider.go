package gcpconsole

import (
	"context"

	"github.com/meigma/whzbox/internal/core/sandbox"
)

// Provider describes Whizlabs' browser-console-only GCP sandbox.
type Provider struct{}

// New returns a GCP console provider.
func New() *Provider { return &Provider{} }

// Kind returns the provider kind.
func (*Provider) Kind() sandbox.Kind { return sandbox.KindGCP }

// Slug returns the Whizlabs task slug.
func (*Provider) Slug() string { return "gcp-sandbox" }

// VerifyCredentials reports that Whizlabs does not supply a
// programmatic GCP credential that can be verified independently.
func (*Provider) VerifyCredentials(context.Context, sandbox.Credentials) (sandbox.Identity, error) {
	return sandbox.Identity{}, sandbox.ErrVerificationUnsupported
}

// Env returns no variables because the GCP sandbox is console-only.
func (*Provider) Env(*sandbox.Sandbox) []string { return nil }
