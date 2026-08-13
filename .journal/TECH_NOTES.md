# Technical Notes

- Whizlabs GCP uses slug `gcp-sandbox` and the lab endpoints `play-create-lab`, `play-update-user-task-status`, and `play-end-lab`, rather than the AWS `play-sandbox` endpoints.
- The observed GCP contract returns a console URL, username/password, project ID/name, and timestamps. It returns no service account, OAuth access token, private key, or other programmatic credential, so `exec gcp` is intentionally unsupported.
- GCP is live-proven only for a one-hour request. Create and teardown can each approach three minutes, so the Whizlabs client uses a four-minute per-request timeout.
- Whizlabs permits one active sandbox per account; `whzbox` blocks creating a different provider while an unexpired provider is cached.
- Whizlabs Azure uses slug `azure-sandbox` and the same lab create, commit, and teardown endpoints as GCP. The live contract returns console credentials, timestamps, and three resource groups, but no programmatic Azure identity.
- Azure requests are live-proven at the one-hour and three-hour bounds. `exec azure` is intentionally unsupported because the response contains no tenant, subscription, service principal, client secret, or equivalent workload credential.
- Azure console login requires an OTP from `play-az-generate-mfa`. `whzbox mfa azure` fetches it only for an unexpired cached Azure sandbox and never persists the code.
- Run local lint with the golangci-lint version pinned in `mise.toml`; an older global binary can miss findings enforced by CI.
