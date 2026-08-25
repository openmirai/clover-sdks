# Changelog

Releases follow [Semantic Versioning](https://semver.org/). Entries are added
under `Unreleased` during review and are promoted by the tag-triggered release
workflow for each `typescript/v*` tag.

## Unreleased

- Adopt Clover V2 transport (`/api/v1`, `CommonResponse` unwrap, `ErrorResponse`).
- Add account/environment-scoped `domains` and `webhooks` resource namespaces;
  platform API-key management remains dashboard-only and is intentionally absent.
- Add Resend drop-in export `@sendclover/sdk/resend` with Result-style responses.
- Auto-send `X-Request-ID` values compatible with Clover DB constraints.
- Add account/environment-scoped `platformMessages` send, get, list, schedule,
  and cancel operations.

## 0.1.0

- Initial public SDK hardening baseline.
