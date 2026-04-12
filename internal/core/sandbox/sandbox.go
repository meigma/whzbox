package sandbox

import "time"

// Kind identifies a sandbox provider type. v1 ships with KindAWS; other
// values are reserved for future providers (GCP, Azure, etc.).
type Kind string

// Known sandbox kinds.
const (
	// KindAWS is the AWS sandbox provided by Whizlabs, surfaced via
	// the whizlabs "aws-sandbox" slug and verified against STS.
	KindAWS Kind = "aws"
)

// Credentials are the secrets handed back by the sandbox broker that
// let the user authenticate to the underlying cloud.
//
// The field set is AWS-centric for v1. Future providers (GCP, Azure)
// will extend this struct with their own equivalents — for example,
// a service-account JSON field for GCP or a client-id/client-secret
// pair for Azure. Kind-specific adapters know which fields apply to
// which kind; unused fields are simply left zero.
type Credentials struct {
	AccessKey string
	SecretKey string
}

// Identity is the "whoami" information verified from the sandbox
// credentials after creation. It is populated by
// Provider.VerifyCredentials and is what gets rendered to the user
// as proof the sandbox is usable.
type Identity struct {
	Account string
	UserID  string
	ARN     string
	Region  string
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
	Identity    Identity
	StartedAt   time.Time
	ExpiresAt   time.Time
}
