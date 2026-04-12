package sandbox

import "errors"

// Sentinel errors returned by the sandbox service and its adapters.
// Callers should use [errors.Is] to test for these; underlying packages
// may wrap them with additional context.
var (
	// ErrUnknownKind is returned when Create is called with a kind
	// that has no registered Provider.
	ErrUnknownKind = errors.New("unknown sandbox kind")

	// ErrProvider wraps sandbox broker or cloud-provider failures.
	ErrProvider = errors.New("sandbox provider error")

	// ErrNoActiveSandbox is returned by Destroy and Status when the
	// user has nothing active. Destroy surfaces this to the caller;
	// Status translates it to (nil, nil).
	ErrNoActiveSandbox = errors.New("no active sandbox")

	// ErrVerificationFailed wraps the underlying credential-check
	// error when Provider.VerifyCredentials rejects the sandbox
	// credentials. The created sandbox is still returned alongside
	// this error so the CLI can surface the credentials to the user
	// anyway — IAM propagation hiccups shouldn't cost them the
	// sandbox.
	ErrVerificationFailed = errors.New("credential verification failed")
)
