# Elixir integration guide (documentation-only)

This is a maintained integration guide, not a supported Clover Elixir SDK. The
former `clover-sdk-elixir` implementation was intentionally removed. Use the
REST contract directly or contribute a separately owned community package.

## Contract-first workflow

Start with [`backend/api/openapi.json`](../../backend/api/openapi.json). Generate
typed Elixir structs/functions with an OpenAPI generator, or define a narrow
adapter from the generated schema. Pin generator and dependency versions; do
not copy stale response shapes into application code.

## Minimal REST shape

```elixir
headers = [
  {~c"authorization", String.to_charlist("Bearer " <> api_key)},
  {~c"idempotency-key", String.to_charlist(idempotency_key)},
  {~c"content-type", ~c"application/json"}
]
body = Jason.encode!(%{from: from, to: to, subject: subject, html: html})
{:ok, {{_, status, _}, response_headers, response_body}} =
  :httpc.request(:post, {String.to_charlist(url), headers, ~c"application/json", body},
    [{:timeout, 30_000}], [{:body_format, :binary}])
```

Use HTTPS and bounded streaming/response accumulation in production. Mutation
keys must be stable 8-128 character ASCII values matching
`[A-Za-z0-9][A-Za-z0-9._:-]{7,127}`; reuse one key for retries. Batch items do
not accept `scheduled_at`.

Success bodies are JSON. Non-2xx bodies are RFC 9457-style
`application/problem+json`; retain unknown members and `x-request-id`. Retry
only safe/idempotent operations, respect non-negative `retry-after` (including
zero), and fail explicitly on malformed or over-limit bodies.

## Community target

Hex publication, Mix CI, and runtime support are not provided by Clover for
this guide. A community implementation must establish independent ownership,
tests, bounded transport, security/release policy, and a separate repository
before it can be evaluated for official support.
