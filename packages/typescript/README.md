# `@sendclover/sdk`

Official TypeScript client for the Clover V2 API (`/api/v1` + `CommonResponse`).

## Install

```sh
npm install @sendclover/sdk
```

## Native client (Clover-shaped)

```ts
import { CloverClient } from "@sendclover/sdk";

const clover = new CloverClient({
  baseUrl: "http://127.0.0.1:8080", // API origin only; paths use /api/v1
  apiKey: process.env.CLOVER_API_KEY!,
});

const accepted = await clover.emails.send(
  {
    from: { address: "sender@example.com" },
    to: [{ address: "user@example.com" }],
    subject: "Hello",
    text: "Sent asynchronously",
  },
  { idempotencyKey: "order-1234" },
);

await clover.domains.list();
await clover.apiKeys.list();
await clover.webhooks.list();
```

The clean platform surface scopes every message call to a client account and
environment:

```ts
const scope = {
  accountId: "account_01",
  environmentId: "environment_01",
};

const message = await clover.platformMessages.send(
  scope,
  {
    from: { address: "sender@example.com" },
    to: [{ address: "user@example.com" }],
    subject: "Hello",
    text: "Sent through the scoped platform API",
  },
  { idempotencyKey: "order-1234" },
);
```

Legacy flat helpers (`send`, `sendBatch`, `schedule`, `cancel`, `get`, `list`)
still work and delegate to `emails.*`.

## Resend drop-in

For apps that already use the Resend Node SDK:

```ts
import { Resend } from "@sendclover/sdk/resend";

const resend = new Resend(process.env.CLOVER_API_KEY, {
  baseUrl: "http://127.0.0.1:8080",
});

const { data, error } = await resend.emails.send({
  from: "Acme <sender@example.com>",
  to: "user@example.com",
  subject: "Hello",
  html: "<p>Hi</p>",
  tags: [{ name: "campaign", value: "welcome" }],
});
```

Supported namespaces: `emails`, `batch`, `domains`, `apiKeys`, `webhooks`.
String addresses, tag arrays, batch arrays, Result-style `{ data, error }`, and
auto `Idempotency-Key` match Resend ergonomics. `emails.update({ id, scheduledAt })`
maps to Clover `POST /api/v1/emails/{id}/schedule`.

Tenants, incidents, and other full-product APIs stay on the native client only.

## Transport notes

- Paths are always `/api/v1/...`
- Success bodies unwrap `CommonResponse.data`
- Errors use V2 `ErrorResponse` (not `problem+json`)
- Mutations require an idempotency key matching
  `^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$` (Resend façade auto-generates)
- Clients send `X-Request-ID` values matching `^req_[A-Za-z0-9_-]{8,128}$`

## Tests

```sh
npm test                 # offline unit/conformance
npm run lint
npm run format:check
```

### Live E2E (opt-in)

Requires the Clover Compose stack (Postgres, Redis, Mailpit, MinIO), migrations,
API, and worker. From the clover repo:

```sh
make up
make migrate
make api      # separate terminal
make worker   # separate terminal
```

Then from `packages/typescript`:

```sh
node scripts/live-e2e-bootstrap.mjs
CLOVER_LIVE_E2E=1 npx vitest run test/e2e.live.test.ts
```

Set `CLOVER_ROOT` if the clover checkout is not a sibling of `clover-sdks`.
Default CI stays offline; live E2E only runs when `CLOVER_LIVE_E2E=1`.

## Optional adapters

`@sendclover/sdk/nestjs` and `@sendclover/sdk/chat` remain optional peer integrations.
See previous sections in this package history for Nest and Chat adapter notes.

Releases use signed `typescript/v*` tags via `release-it`; see
[`RELEASING.md`](RELEASING.md).
