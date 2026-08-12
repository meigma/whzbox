package sandbox

import "time"

// Kind identifies a sandbox provider type.
type Kind string

// Known sandbox kinds.
const (
	// KindAWS is the AWS sandbox provided by Whizlabs, surfaced via
	// the whizlabs "aws-sandbox" slug and verified against STS.
	KindAWS Kind = "aws"

	// KindGCP is the GCP sandbox provided by Whizlabs, surfaced via the
	// "gcp-sandbox" slug with browser-console credentials.
	KindGCP Kind = "gcp"

	// KindAzure is the Azure sandbox provided by Whizlabs, surfaced via
	// the "azure-sandbox" slug with browser-console credentials.
	KindAzure Kind = "azure"
)

// Credentials are the secrets handed back by the sandbox broker that
// let the user authenticate to the underlying cloud.
//
// The field set contains AWS's programmatic credentials. Providers
// without programmatic credentials leave these fields empty.
type Credentials struct {
	AccessKey string
	SecretKey string
}

// Identity contains provider-specific sandbox metadata. AWS populates it
// through credential verification; console-only providers populate it from
// the Whizlabs create response.
type Identity struct {
	Account     string
	UserID      string
	ARN         string
	Region      string
	ProjectID   string
	ProjectName string
	// ResourceGroups are the Azure resource groups assigned to the sandbox.
	ResourceGroups []string
}

// Console is the browser-login information returned alongside
// programmatic credentials. Whizlabs provides a console URL plus a
// username/password pair for every sandbox kind.
type Console struct {
	URL      string
	Username string
	Password string
}

// Sandbox is the top-level domain value for a provisioned environment.
// It bundles everything a user needs to actually use the sandbox plus
// metadata about when it expires.
//
// Identity is empty until Provider.VerifyCredentials runs.
type Sandbox struct {
	Kind        Kind
	Slug        string // whizlabs slug, e.g. "aws-sandbox"
	Credentials Credentials
	Console     Console
	// Verified reports whether the provider-specific credential check
	// completed successfully for this sandbox.
	Verified  bool
	Identity  Identity
	StartedAt time.Time
	ExpiresAt time.Time
}
