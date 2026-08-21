# Clover SDKs

Official Clover API clients live in this polyglot monorepo. The current release
scope is intentionally limited to the Go SDK and Go-based Clover CLI. Other
language directories are preserved as deferred implementation work and are not
part of the current support or readiness claim.

## Current release scope

| Platform | Location | Distribution |
| --- | --- | --- |
| Go | [`packages/go`](packages/go/) | Go module `github.com/openmirai/clover-sdks/packages/go` |
| CLI | [`apps/cli`](apps/cli/) | GoReleaser binaries and Go module `github.com/openmirai/clover-sdks/apps/cli` |

TypeScript, Python, Java/Kotlin, Rust, Swift, and Dart/Flutter packages are
deferred. Their presence in this repository is not a publication, support, or
compatibility guarantee. Documentation-only community targets for .NET,
Elixir, PHP, and Ruby are in [`docs/community`](docs/community/).

## V2 + Resend layering

The Go SDK and CLI target Clover V2:

- Base path **`/api/v1`**
- Success envelope **`CommonResponse`** (SDKs unwrap `data`)
- Errors as **`ErrorResponse`** (not `application/problem+json`)

See [`docs/CONTRACT.md`](docs/CONTRACT.md) for the full contract.

## Development

Use the native toolchain inside the package you are changing. The root
`Makefile` provides the same gates used by CI:

```sh
make check
```

`make check` validates the repository layout, Go SDK, and CLI. The `check-all`
target remains available for future work on deferred package directories, but
it is not part of the current readiness gate.

## Versioning and releases

Current releases use independent tags:

- `packages/go/vX.Y.Z`
- `apps/cli/vX.Y.Z`

Deferred package release workflows and metadata are retained for later
completion, but are outside the current release approval.

## Compatibility contract

The Go SDK exposes the Phase 1 resource contract. Mutation calls require a
bounded `Idempotency-Key`; the SDK preserves request metadata, bounds response
bodies, and retries only safe transient failures. Sync OpenAPI from the Clover
backend swagger into `openapi/`.
