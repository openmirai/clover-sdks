# Changelog

Releases follow [Semantic Versioning](https://semver.org/). Entries are added
under `Unreleased` during review and are promoted by `release-it` for each
signed `v*` tag.

## Unreleased

- Adopt Clover V2 transport (`/api/v1`, `CommonResponse` unwrap, `ErrorResponse`).
- Add native `domains`, `apiKeys`, and `webhooks` resource namespaces.
- Add Resend drop-in export `@sendclover/sdk/resend` with Result-style responses.
- Add opt-in live Compose E2E (`CLOVER_LIVE_E2E=1`).
- Auto-send `X-Request-ID` values compatible with Clover DB constraints.
- Add account/environment-scoped `platformMessages` send, get, list, schedule,
  and cancel operations.

## 0.1.0

- Initial public SDK hardening baseline.
