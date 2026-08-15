# `@clover/sdk`

Small, dependency-free TypeScript client for the Clover API. It uses the
platform `fetch` implementation, sends bearer authentication and a
`User-Agent`, and requires an `Idempotency-Key` for every mutation.

```ts
import { CloverClient } from "@clover/sdk";
const clover = new CloverClient({ baseUrl: "https://api.example.com", apiKey: process.env.CLOVER_API_KEY! });
const accepted = await clover.send({
  from: { address: "sender@example.com" }, to: [{ address: "user@example.com" }],
  subject: "Hello", text: "Sent asynchronously",
}, "order-1234");
```

`send`, `sendBatch`, `schedule`, `cancel`, `get`, and cursor-aware `list` are
provided. `CloverError.problem` retains unknown problem members. GET requests,
and idempotent mutations, retry at most three times for transient statuses;
`Retry-After` is honored when present. Inject `fetch` and `sleep` for tests.

Run the deterministic offline checks with `npm test`, `npm run lint`, and
`npm run format:check`. Development uses Vitest and the Oxc toolchain
(oxlint/oxfmt); `npm run test:coverage` writes a local coverage report.

Releases are made from signed `typescript/v*` tags by `release-it`; see
[`RELEASING.md`](RELEASING.md). Contributions and vulnerability reports are
covered by [`CONTRIBUTING.md`](CONTRIBUTING.md) and [`SECURITY.md`](SECURITY.md).

## Optional adapters

`@clover/sdk/nestjs` exports a structural `CloverNestModule.forRoot()` dynamic
module/provider adapter. Install `@nestjs/common` in the application when
using it; Nest is an optional peer and is never loaded by the core client.

`@clover/sdk/chat` provides a typed, framework-free Clover Chat adapter core
for Vercel's optional `chat` peer. It maps RFC `Message-ID`, `In-Reply-To`, and
`References` headers, exposes an application-owned webhook signature verifier,
and implements outbound `openDM`/`post`. Edit, delete, reaction, and typing
operations fail explicitly because Clover does not provide those semantics.
Pass an explicit `baseUrl` or an already-configured `CloverClient`; no endpoint
is assumed. The exact Vercel Chat SDK bridge remains application-owned so this
package does not force a framework dependency.
