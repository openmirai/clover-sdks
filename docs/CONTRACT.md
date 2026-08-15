# Clover SDK ecosystem contract

Verified: 2026-08-15

## Coverage

Resend currently lists Node.js, PHP, Laravel, Python, Ruby, Go, Java, Rust, and
.NET as official SDKs, plus community SDKs for Elixir, NestJS, and Dart. Clover
maintains one official monorepo for the supported client set below; PHP,
Laravel, Ruby, .NET, and Elixir remain documentation-only community targets:

| Ecosystem | Clover repository | Integration notes |
| --- | --- | --- |
| Node.js / TypeScript / NestJS / Chat SDK | `packages/typescript` | ESM package exports; optional NestJS provider and Vercel Chat SDK adapter subpath |
| Python | `packages/python` | Typed sync client |
| Go | `packages/go` | Go module |
| Java / Kotlin JVM | `packages/java` | Java client usable from Kotlin/JVM |
| Rust | `packages/rust` | crates.io package |
| Dart / Flutter | `packages/dart` | pub.dev package |
| Swift / Apple platforms | `packages/swift` with root `Package.swift` | Swift Package Manager package |
| CLI | `apps/cli` | GoReleaser multi-platform binary |
| OpenAPI | `openapi/clover-v1.json` | Canonical generated contract snapshot |

Documentation-only community guides for the deferred ecosystems are maintained
at [`docs/community/`](community/). They define contract-first REST
usage and generated-client options but make no package, framework, or support
claim.

The Swift SDK is an approved additional Clover platform lane. Kotlin uses the
Java/JVM artifact rather than duplicating the client, and Flutter consumes the
Dart package. The Vercel Chat SDK adapter is an optional TypeScript export so it
does not create a second repository for the same language ecosystem.

Sources:

- <https://resend.com/docs/sdks>
- <https://resend.com/open-source>

## Required client behavior

Every SDK exposes the canonical send, batch send, schedule, cancel, get, and
list operations when the public Clover contract supports them. Every client:

- authenticates with a bearer API key and identifies its language/version in
  `User-Agent`;
- validates required idempotency keys before a mutation leaves the process;
- parses `application/problem+json` without discarding unknown extension
  fields;
- percent-encodes path segments and query values correctly;
- returns request/response metadata needed for support correlation;
- retries only safe/idempotent operations, with bounded attempts, backoff, and
  `Retry-After` support;
- has deterministic transport-injected conformance tests and performs no live
  sends during CI.

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

## TypeScript baseline

The TypeScript SDK must use:

- `release-it` for versioning and tag/release orchestration;
- Vitest for unit/conformance tests;
- Oxlint and Oxfmt for linting and formatting;
- lint-staged with a Git pre-commit hook;
- strict TypeScript declarations, package export maps, provenance-enabled npm
  publishing, and supported Node.js CI versions.

## New-language baselines

- Swift: Swift Package Manager, XCTest, swift-format, SwiftLint, DocC-ready
  public API, Apple/Linux CI where supported, semantic tags.
- Dart: pub package, `dart format`, analyzer, `dart test`, `dart pub publish
  --dry-run`, and tag-gated pub.dev publishing.
