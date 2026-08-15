# Clover SDK ecosystem readiness

Status: implementation-aligned inventory
Date: 2026-08-15

The canonical client contract snapshot is
[`openapi/clover-v1.json`](../openapi/clover-v1.json). The checkout contains one
independently publishable directory for each maintained language
runtime plus the CLI. Maintained clients implement the common send, batch,
schedule, cancel, get, and list lifecycle where the API operation exists; they
validate mutation idempotency keys before transport, preserve unknown RFC 9457
problem members, return correlation metadata, enforce a configurable 4 MiB
response limit, and bound retries to safe/idempotent requests. PHP/Laravel,
Ruby, .NET, and Elixir are documentation-only community targets, not SDK
implementations.

## Inventory

| Ecosystem | Directory | Readiness in this checkout |
| --- | --- | --- |
| Node.js / TypeScript / NestJS / Vercel Chat | [`packages/typescript/`](../packages/typescript/) | ESM client with fetch injection, `./nestjs` dynamic module/provider and `./chat` adapter exports, optional peer dependencies, Vitest/Oxlint/Oxfmt, release-it, and lifecycle/integration conformance tests. `npm test` passes locally (10 tests); the coverage gate records 92.17% statements and 90% functions. |
| Python | [`packages/python/`](../packages/python/) | Python 3.10+ urllib client with pytest/ruff/mypy configuration and offline tests. |
| Go | [`packages/go/`](../packages/go/) | Standard-library Go module with injected `Doer`/sleep and conformance tests. |
| Java / Kotlin JVM | [`packages/java/`](../packages/java/) | Java 17 Maven artifact usable from Kotlin/JVM with a reusable `HttpClient`, JUnit conformance coverage, and signed Central Publisher Portal release metadata. |
| Rust | [`packages/rust/`](../packages/rust/) | Rust 2021 Cargo library with strict JSON handling and offline transport tests. |
| Swift / Apple platforms | [`packages/swift/`](../packages/swift/) | Root Swift Package Manager package with async/await, injectable transport, XCTest, swift-format, and SwiftLint configuration. `swift test` passes locally. |
| Dart / Flutter | [`packages/dart/`](../packages/dart/) | pub package with typed async client, analyzer, formatter, and Dart tests. Dart/Flutter are not installed here; CI owns those checks. |
| CLI | [`apps/cli/`](../apps/cli/) | Go CLI with send, send-batch, schedule, get, filtered list, cancel, and bounded SSE stream commands plus injected HTTP tests. |

The maintained Swift and Dart directories expose the same lifecycle contract;
Swift has local test/lint evidence and Dart requires CI or a matching developer
toolchain. Documentation-only community guides for PHP/Laravel, Ruby, .NET,
and Elixir live under [`docs/community/`](community/).

## Ecosystem extensions

The TypeScript NestJS dynamic module/provider plus Vercel Chat adapter exports
are present with native integration tests. The Vercel Chat bridge remains
application-owned and framework peers are optional, so the core client stays
dependency-free. Kotlin and Flutter consume the Java/JVM and Dart artifacts
respectively; they do not require duplicate repositories. PHP/Laravel, Ruby,
.NET, and Elixir guides document REST/OpenAPI integration and generated-client
options without claiming framework support.

## Local evidence and release boundary

The package manifests, tests, and root workflows are the current readiness
evidence. Historical implementation handoffs remain in the Clover platform
workspace and do not expand the maintained set.
Every release workflow is SemVer tag-gated and fails closed unless the tag
matches package metadata (or, for Go modules and Swift packages, the exact
tag-derived source). No SDK test performs a production send, uses a customer
secret, or proves provider delivery. Package publication, trusted-publishing
credentials, unavailable
Maven/Dart toolchains, live SES/SMTP behavior, DNS, deployment, and UAT remain
external release gates. The documentation-only community guides have no
publication or support gate in this repository.
