# Clover SDK ecosystem contract

Verified: 2026-08-20

## Coverage

Resend maintains a broad SDK ecosystem. Clover's current supported client
contract is intentionally narrower: the Go SDK and the Go-based CLI are the
only release-ready clients in this milestone.

| Ecosystem | Clover repository | Integration notes |
| --- | --- | --- |
| Go | `packages/go` | Go module |
| CLI | `apps/cli` | GoReleaser multi-platform binary |
| OpenAPI | `openapi/clover-v1.json` | Byte-synced from backend `cmd/api/docs/swagger.json`; verify with `make check-openapi` |

TypeScript, Python, Java/Kotlin, Rust, Swift, and Dart/Flutter package
directories are deferred and carry no current support, compatibility, or
publication claim. Documentation-only community guides are maintained at
[`docs/community/`](community/).

## Deferred TypeScript design notes

This section records future design intent only. It is not part of the current
support or readiness claim.

1. **Native** — `import { CloverClient } from "@clover/sdk"`
   Clover-shaped API over `/api/v1`, unwraps `CommonResponse`, parses
   `ErrorResponse`, and exposes `emails`, `domains`, `apiKeys`, and `webhooks`
   (plus legacy flat email helpers). Full-product surfaces such as tenants and
   incidents stay off the Resend façade.
2. **Resend drop-in** — `import { Resend } from "@clover/sdk/resend"`
   Mirrors the Resend Node surface for overlapping ops (`emails`, `batch`,
   `domains`, `apiKeys`, `webhooks`): string addresses, tag arrays, batch raw
   arrays → `{ items }`, Result-style `{ data, error }`, auto
   `Idempotency-Key`, and `emails.update({ scheduledAt })` → Clover
   `POST .../schedule`.

## Required client behavior

The Go SDK exposes the canonical Phase 1 operations supported by the public
Clover contract. The Go SDK and CLI:

- authenticates with a bearer API key and identifies its language/version in
  `User-Agent`;
- calls **`/api/v1/...`** (not `/v1/...`) and unwraps `CommonResponse.data` on
  success;
- parses V2 `ErrorResponse` (`success: false`, `error.code/type/message`) —
  not RFC 9457 `application/problem+json`;
- sends `X-Request-ID` values matching `^req_[A-Za-z0-9_-]{8,128}$` (auto when
  omitted);
- validates required idempotency keys before a mutation leaves the process
  (Resend façade may auto-generate);
- percent-encodes path segments and query values correctly;
- returns request/response metadata needed for support correlation;
- retries only safe/idempotent operations, with bounded attempts, backoff, and
  `Retry-After` support;
- has deterministic transport-injected conformance tests and performs no live
  sends during default CI.

## Go SDK Phase 1 platform surface

The account/environment-scoped Phase 1 surface is exposed under
`client.Platform`. New resources use canonical paths with escaped
`account_id` and `environment_id` segments; they do not accept or emit the
legacy `tenant_id`, `project_id`, or query-style `environment` scope.

- `Platform.Accounts` and `Platform.Environments` manage client accounts and
  environments.
- `Platform.Messages` covers transactional send, batch, get, schedule, and
  cancel. `Platform.Templates` covers immutable versions, compare, render,
  publish, rollback, archive, and unarchive.
- `Platform.Webhooks` and `Platform.Timeline` cover scoped webhook delivery
  and redacted message history. `Platform.Inbound` covers received messages,
  signed provider callbacks, receiving-domain configuration, and attachments.
  `Platform.Preferences` and `Platform.Suppressions` cover hosted preference
  tokens, topics, RFC 8058 unsubscribe, and scoped consent.
- `Platform.SMTP` covers scoped credentials and submission history.
  `Platform.Contacts`, `Platform.Segments`, and `Platform.Automations` cover
  the audience/lifecycle foundation.
- `Platform.Routing` covers policy, providers, routes, pools, IP assignments,
  and deterministic dry-run resolution. `Platform.Domains` and
  `Platform.DomainHealth` cover outbound-domain onboarding and authentication
  reports. `Platform.Usage` covers vocabulary, immutable fact export and
  corrections, and reconciliation evidence.

Every environment-scoped method takes `PlatformScope{AccountID,
EnvironmentID}`. Mutations documented with an `Idempotency-Key` validate it
before transport, and the client bounds both request and response bodies.
GETs and explicitly keyed mutations may retry transient responses with bounded
backoff and `Retry-After`; unkeyed writes are not retried. Response metadata
retains correlation and replay/rate-limit details, while secret-bearing
mutation bodies are never retained.

## Live E2E (opt-in)

Offline unit/conformance tests remain the default CI gate. Live Compose E2E is
gated behind `CLOVER_LIVE_E2E=1`. See the root README and
`packages/typescript/README.md` for bootstrap steps against the Clover stack.

## Repository readiness

The monorepo and each independently published package must include:

1. Ecosystem-standard manifest with version, license, repository, homepage,
   minimum runtime/platform, and package contents.
2. Formatter, linter, static analysis, unit tests, package/build validation, and
   a single local `check` entry point.
3. Pre-commit checks scoped to changed files where the ecosystem supports it.
4. GitHub Actions CI for supported runtime versions and a tag-gated release
   workflow using trusted publishing or repository secrets; release jobs must
   never publish on pull requests.
5. `CHANGELOG.md`, `CONTRIBUTING.md`, `SECURITY.md`, full MIT `LICENSE`, and an
   ignore file that excludes credentials and build output.
6. No committed tokens, live endpoints in tests, generated build artifacts, or
   network-dependent unit tests.

## Deferred TypeScript baseline

Before future support, the TypeScript SDK must use:

- `release-it` for versioning and tag/release orchestration;
- Vitest for unit/conformance tests;
- Oxlint and Oxfmt for linting and formatting;
- lint-staged with a Git pre-commit hook;
- strict TypeScript declarations, package export maps, provenance-enabled npm
  publishing, and supported Node.js CI versions.

## Deferred language baselines

- Swift: Swift Package Manager, XCTest, swift-format, SwiftLint, DocC-ready
  public API, Apple/Linux CI where supported, semantic tags.
- Dart: pub package, `dart format`, analyzer, `dart test`, `dart pub publish
  --dry-run`, and tag-gated pub.dev publishing.
