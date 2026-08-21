# Clover Go SDK and CLI readiness

Status: local implementation evidence
Date: 2026-08-21

The current release boundary contains only:

| Surface | Directory | Readiness claim |
| --- | --- | --- |
| Go SDK | [`packages/go/`](../packages/go/) | Typed Phase 1 resource client with injected transport, bounded retries and response bodies, idempotency validation, request metadata, and race-tested conformance coverage. |
| Clover CLI | [`apps/cli/`](../apps/cli/) | Go CLI over the same Phase 1 resource contract, including provider-neutral routing controls and local development workflows. |
| OpenAPI | [`openapi/clover-v1.json`](../openapi/clover-v1.json) | Snapshot synchronized from the backend contract before release verification. |

TypeScript, Python, Java/Kotlin, Rust, Swift, and Dart/Flutter directories are
deferred. They are not included in the current CI, support, compatibility, or
release-readiness claim. Their package metadata and release workflows are kept
as future implementation scaffolding only. Documentation-only community guides
also make no SDK support claim.

## Local evidence and external gates

`make check` is the required local gate for repository layout, the Go SDK, and
the CLI. Default tests use injected transports and do not send customer email.
Publishing a version, pushing its tag, running remote CI, authenticating to a
live Clover deployment, and proving delivery through a live provider remain
external release gates.
