# Technical Notes

- Whizlabs GCP uses slug `gcp-sandbox` and the lab endpoints `play-create-lab`, `play-update-user-task-status`, and `play-end-lab`, rather than the AWS `play-sandbox` endpoints.
- The observed GCP contract returns a console URL, username/password, project ID/name, and timestamps. It returns no service account, OAuth access token, private key, or other programmatic credential, so `exec gcp` is intentionally unsupported.
- GCP is live-proven only for a one-hour request. Create and teardown can each approach three minutes, so the Whizlabs client uses a four-minute per-request timeout.
- Whizlabs permits one active sandbox per account; `whzbox` blocks creating a different provider while an unexpired provider is cached.
