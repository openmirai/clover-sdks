# Clover Go SDK

`github.com/openmirai/clover-sdks/packages/go` is a dependency-free Go client for the
Clover API. Configure `NewClient(baseURL, apiKey)`, then call `Send`,
`SendBatch`, `Schedule`, `Cancel`, `Get`, or `List`. Mutation methods require
an idempotency key and all requests include bearer auth and a user-agent.

```go
client := clover.NewClient("https://api.example.com", os.Getenv("CLOVER_API_KEY"))
accepted, _, err := client.Send(ctx, clover.JSON{
    "from": map[string]any{"address": "sender@example.com"},
    "to": []any{map[string]any{"address": "user@example.com"}},
    "subject": "Hello", "text": "Accepted asynchronously",
}, "order-1234")
```

The client decodes `application/problem+json` into `*clover.Error` and keeps
unknown problem fields in `Problem.Extra`. Response metadata exposes request
ID, replay, rate-limit, and retry-after headers. GETs and idempotent mutations
retry transient responses at most three times. Inject `HTTPClient` and `Sleep`
for deterministic tests.

Run `go test ./...`.
