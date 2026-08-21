# Clover Go SDK

`github.com/openmirai/clover-sdks/packages/go` is a dependency-free Go client for the
Clover API. Configure `NewClient(baseURL, apiKey)`. The original flat email
helpers remain supported; the typed, namespaced services are available on the
same client (`Emails`, `Domains`, `Templates`, `Webhooks`, `APIKeys`, `Logs`,
`Metrics`, `Inbound`, `Suppressions`, `Preferences`, `Contacts`, `Segments`,
`Broadcasts`, `Automations`, `Audit`, `DomainHealth`, and `Routing` (also
available as `ProviderRouting`).

```go
client := clover.NewClient("https://api.example.com", os.Getenv("CLOVER_API_KEY"))
accepted, _, err := client.Emails.Send(ctx, clover.SendEmailRequest{
    From: clover.EmailAddress{Address: "sender@example.com"},
    To: []clover.EmailAddress{{Address: "user@example.com"}},
    Subject: "Hello", Text: clover.String("Accepted asynchronously"),
}, "order-1234")
```

For tokenized preference-center and one-click unsubscribe links, use
`NewPublicClient(baseURL)`; it omits the bearer header and still enforces
idempotency keys only when the backend contract requires them. Public
preference updates and one-click unsubscribe are intentionally keyless.

Typed resource methods map directly to the canonical `/api/v1` Swagger paths:

- Domains and DNS verification: `Domains.List/Create/Get/Update/Delete`,
  `DNSRecords`, `ProvisionDNS`, and `Verify`; `DomainHealth.List/Get/Verify`
  covers the full-product DNS health reports.
- Templates and immutable versions: `Templates.List/Create/Get/Update`,
  `Versions`, `CreateVersion`, `GetVersion`, `Compare`, `Publish`, and
  `Rollback` (the explicit `/rollback` route). Version artifacts and compare
  diffs are represented by typed fields.
- Webhooks and replay: endpoint CRUD, secret rotation, delivery list/detail,
  and `ReplayDelivery`.
- Email logs/events and inbound mail: `Emails.Trace/Replay`, `Logs.List`,
  `Metrics.Email`, `Inbound.List/Get/AttachmentURL`, and provider callback
  helpers.
- Audience and lifecycle: contacts (including `Resubscribe`), segments and
  evaluation, broadcasts, and automations/runs/events.
- Governance and consent: scoped API keys, suppressions/reactivation,
  preference topics, tokenized preference centers and one-click unsubscribe,
  and append-only audit events/holds.
- Provider-neutral routing: organization policy, provider capabilities,
  dedicated IP pools and IPs, warmup plans, lifecycle commands, and routing
  transition audit.

Methods backed by Clover's idempotency middleware require a valid key and
reject missing keys before transport. Routes without that middleware (for
example API-key, domain, webhook, suppression, and preference writes) accept an
optional key and do not invent a requirement. Provider-routing mutations also
require a key because the routing service uses it as a durable command ID even
though its HTTP handler does not install middleware. Native
full-product lifecycle operations require `project_id`, `environment`, and
`tenant_id` in their `ScopedPageOptions`; aggregate domain-health, email
reliability, and audit scopes allow tenant omission where the backend does.
Invalid scopes fail before transport. GETs and the side-effect-free segment
evaluation are retry-safe; keyed mutations are retry-safe as well. Query
values and path segments are percent-encoded, successful `CommonResponse.data`
envelopes are unwrapped into typed values, and `*clover.Error` exposes status,
structured error details, correlation metadata, and a bounded `RawBody` only
for read responses. Secret-bearing mutation bodies (including one-time API-key
tokens) are never retained in response metadata.

The client decodes Clover `ErrorResponse` (and legacy problem bodies) into
`*clover.Error` and keeps unknown problem fields in `Problem.Extra`. Response
metadata exposes request ID, replay, rate-limit, retry-after, and the bounded
raw body. GETs and idempotent mutations retry transient responses at most three
times. Inject `HTTPClient` and `Sleep` for deterministic tests.

Run `gofmt -w .`, `go vet ./...`, and `go test -race ./...`.
