# Clover SDKs

Official Clover API clients live in this polyglot monorepo. Each package keeps
its native build, test, lint, formatting, and publishing conventions while CI,
security policy, dependency updates, and repository governance are managed once
at the root.

## Maintained clients

| Platform | Location | Distribution |
| --- | --- | --- |
| TypeScript / Node.js | [`packages/typescript`](packages/typescript/) | npm `@clover/sdk` (+ `@clover/sdk/resend`) |
| Python | [`packages/python`](packages/python/) | PyPI `clover-sdk` |
| Go | [`packages/go`](packages/go/) | Go module `github.com/openmirai/clover-sdks/packages/go` |
| Java / Kotlin JVM | [`packages/java`](packages/java/) | Maven `dev.clover:clover-sdk` |
| Rust | [`packages/rust`](packages/rust/) | crates.io `clover-sdk` |
| Swift / Apple platforms | [`packages/swift`](packages/swift/) | Swift Package Manager from the repository root |
| Dart / Flutter | [`packages/dart`](packages/dart/) | pub.dev `clover_sdk` |
| CLI | [`apps/cli`](apps/cli/) | GoReleaser binaries and Go module `github.com/openmirai/clover-sdks/apps/cli` |

Documentation-only community targets for .NET, Elixir, PHP, and Ruby are in
[`docs/community`](docs/community/). They are not maintained implementations.

## V2 + Resend layering

All clients target Clover V2:

- Base path **`/api/v1`**
- Success envelope **`CommonResponse`** (SDKs unwrap `data`)
- Errors as **`ErrorResponse`** (not `application/problem+json`)

TypeScript ships two surfaces:

- **Native** `@clover/sdk` — Clover-shaped API + domains / apiKeys / webhooks
- **Resend drop-in** `@clover/sdk/resend` — `new Resend(apiKey)` with Result-style
  `{ data, error }` for overlapping email/domain/key/webhook ops

See [`docs/CONTRACT.md`](docs/CONTRACT.md) for the full contract.

## Development

Use the native toolchain inside the package you are changing. The root
`Makefile` provides the same gates used by CI:

```sh
make check
```

`make check` requires every maintained toolchain. Individual targets such as
`make check-typescript`, `make check-go`, or `make check-swift` are available for
focused work. Root pre-commit hooks are path-scoped and delegate to native
formatters and linters.

### Live TypeScript E2E

Default CI is offline. Against a local Clover Compose stack:

```sh
# in clover/
make up && make migrate
make api    # terminal 1
make worker # terminal 2

# in clover-sdks/packages/typescript/
node scripts/live-e2e-bootstrap.mjs
CLOVER_LIVE_E2E=1 npx vitest run test/e2e.live.test.ts
```

## Versioning and releases

Packages version independently. A release tag must match its package manifest:

- `typescript/vX.Y.Z`
- `python/vX.Y.Z`
- `packages/go/vX.Y.Z`
- `java/vX.Y.Z`
- `rust/vX.Y.Z`
- `dart/vX.Y.Z`
- `apps/cli/vX.Y.Z`
- `vX.Y.Z` for the root Swift package

Release workflows validate the tag and package version before publishing. They
use registry-native trusted publishing or scoped release credentials where the
registry supports them.

## Compatibility contract

All clients expose the canonical send, batch-send, schedule, cancel, get, and
list operations. Mutation calls require a bounded `Idempotency-Key`; clients
preserve request metadata, bound response bodies, and retry only safe transient
failures. Sync OpenAPI from the Clover backend swagger into `openapi/`.
