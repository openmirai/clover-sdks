# Clover SDK and CLI readiness

Status: local implementation evidence
Date: 2026-08-21

The current release boundary contains only:

| Surface | Directory | Readiness claim |
| --- | --- | --- |
| TypeScript SDK | [`packages/typescript/`](../packages/typescript/) | Typed V2 client plus account/environment-scoped platform messages, request IDs, idempotency validation, generated declarations, and Vitest transport conformance. |
| Go SDK | [`packages/go/`](../packages/go/) | Typed Phase 1 resource client with injected transport, bounded retries and response bodies, idempotency validation, request metadata, and race-tested conformance coverage. |
| Clover CLI | [`apps/cli/`](../apps/cli/) | Go CLI over the same Phase 1 resource contract, including provider-neutral routing controls and local development workflows. |
| OpenAPI | [`openapi/clover-v1.json`](../openapi/clover-v1.json) | Snapshot synchronized from the backend contract before release verification. |

Python, Java/Kotlin, Rust, Swift, and Dart/Flutter directories are not included
in the current publication claim. Their package metadata and release workflows
remain integration scaffolding until their native toolchain and registry gates
are complete. Documentation-only community guides also make no SDK support
claim.

## Local evidence and external gates

`make check` is the required local gate for repository layout, the Go SDK, and
the CLI. The TypeScript package adds formatter, linter, strict typecheck,
Vitest, build, and npm-pack gates. Default tests use injected transports and do
not send customer email.
Publishing a version, pushing its tag, running remote CI, authenticating to a
live Clover deployment, and proving delivery through a live provider remain
external release gates.
