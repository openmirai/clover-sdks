# `@sendclover/sdk`

Official TypeScript client for Clover's account/environment-scoped platform API.
Read the [Clover API reference](https://staging.sendclover.com/en/api-reference)
for the public request and response contract.

## Install

```sh
npm install @sendclover/sdk
```

## Native client

Bind the client to a Clover account and environment. All resource calls then use
the current `/api/v1/platform/...` route family and camelCase DTOs.

```ts
import { CloverClient } from "@sendclover/sdk";

const clover = new CloverClient({
  baseUrl: process.env.CLOVER_API_URL!,
  apiKey: process.env.CLOVER_API_KEY!,
  accountId: process.env.CLOVER_ACCOUNT_ID!,
  environmentId: process.env.CLOVER_ENVIRONMENT_ID!,
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
await clover.webhooks.list();
```

For applications that select several environments dynamically, use the
explicit platform message resource:

```ts
const message = await clover.platformMessages.send(
  { accountId: "account_01", environmentId: "environment_01" },
  {
    from: { address: "sender@example.com" },
    to: [{ address: "user@example.com" }],
    subject: "Hello",
    text: "Sent through the scoped platform API",
  },
  { idempotencyKey: "order-1234" },
);
```

The client exposes scoped `emails`, `domains`, and `webhooks` resources plus
top-level email convenience methods. Platform API-key management stays in the
dashboard-only control plane and is intentionally not exposed here.

## Resend-compatible façade

For applications using Resend-style method names, pass the same account and
environment scope:

```ts
import { Resend } from "@sendclover/sdk/resend";

const resend = new Resend(process.env.CLOVER_API_KEY, {
  baseUrl: process.env.CLOVER_API_URL,
  accountId: process.env.CLOVER_ACCOUNT_ID,
  environmentId: process.env.CLOVER_ENVIRONMENT_ID,
});

const { data, error } = await resend.emails.send({
  from: "Acme <sender@example.com>",
  to: "user@example.com",
  subject: "Hello",
  html: "<p>Hi</p>",
  tags: [{ name: "campaign", value: "welcome" }],
});
```

The façade maps email and batch sends, scheduling, cancellation, domains, and
webhooks to the scoped platform routes. Domain onboarding requires
`providerBindingId`. Results use `{ data, error }`, and mutations automatically
send a valid `Idempotency-Key` when one is not supplied.

## Transport contract

- `baseUrl` is the API origin only; paths are scoped under `/api/v1/platform`.
- Authentication uses `Authorization: Bearer <api-key>`.
- Success bodies unwrap `CommonResponse.data`.
- Errors use the V2 nested `ErrorResponse` envelope.
- Mutations require an idempotency key matching
  `^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$`.
- Clients send `X-Request-ID` values matching `^req_[A-Za-z0-9_-]{8,128}$`.

## Quality gates

```sh
npm run quality       # lint, format, typecheck, tests, coverage thresholds
npm pack --dry-run    # inspect the publish file list
```

The package also runs `quality` from `prepublishOnly`. Provider and DNS
qualification stays in Clover's backend-owned E2E harness because it requires
real provider bindings and public DNS; the SDK publication gate remains fully
offline and deterministic.
