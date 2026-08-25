# Clover SDKs

Official Clover API clients live in this polyglot monorepo. The current release
scope covers the TypeScript SDK, Go SDK, and Go-based Clover CLI. Other language
directories remain integration-ready previews until their package-specific
release gates are complete.

## Current release scope

| Platform | Location | Distribution |
| --- | --- | --- |
| TypeScript | [`packages/typescript`](packages/typescript/) | npm package `@sendclover/sdk` |
| Go | [`packages/go`](packages/go/) | Go module `github.com/openmirai/clover-sdks/packages/go` |
| CLI | [`apps/cli`](apps/cli/) | GoReleaser binaries and Go module `github.com/openmirai/clover-sdks/apps/cli` |

Python, Java/Kotlin, Rust, Swift, and Dart/Flutter packages are not included in
the current publication claim. Documentation-only community targets for .NET,
Elixir, PHP, and Ruby are in [`docs/community`](docs/community/).

## V2 + Resend layering

The TypeScript SDK, Go SDK, and CLI target Clover V2:

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

`make check` validates the repository layout, Go SDK, and CLI. TypeScript is
verified independently with `npm run format:check`, `npm run lint`,
`npm run typecheck`, `npm test`, and `npm pack --dry-run` in its package.

## Versioning and releases

Current releases use independent tags:

- `typescript/vX.Y.Z`
- `packages/go/vX.Y.Z`
- `apps/cli/vX.Y.Z`

Deferred package release workflows and metadata are retained for later
completion, but are outside the current release approval.

## Compatibility contract

The Go SDK exposes the Phase 1 resource contract. Mutation calls require a
bounded `Idempotency-Key`; the SDK preserves request metadata, bounds response
bodies, and retries only safe transient failures. Sync OpenAPI from the Clover
backend swagger into `openapi/`.
