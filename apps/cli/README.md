# clover-cli

Dependency-free Clover command-line client. Set `CLOVER_BASE_URL` and
`CLOVER_API_KEY`, then use:

```text
clover-cli send -from sender@example.com -to user@example.com -subject Hello -text Body -idempotency-key order-1234
clover-cli send-batch -items '[{"subject":"Hello"}]' -idempotency-key batch-1234
clover-cli schedule <email-id> -scheduled-at 2030-01-01T00:00:00Z -idempotency-key schedule-1234
clover-cli get <email-id>
clover-cli list -status queued -limit 50 -cursor next-page
clover-cli cancel <email-id> -idempotency-key cancel-1234
clover-cli stream [/v1/events/stream]
```

`stream` consumes newline-delimited SSE `data:` events until the server closes
the connection and prints each JSON event. The shared client sends bearer auth,
user-agent, and idempotency headers, decodes problem JSON while retaining
unknown fields, and retries only safe requests or idempotent mutations with a
maximum of three attempts. Tests inject an HTTP round-tripper; no real network
is needed.

Build or test with `go test ./...` and install with
`go install github.com/openmirai/clover-sdks/apps/cli@latest`.
