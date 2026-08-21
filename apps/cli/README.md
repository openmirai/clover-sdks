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
clover-cli dev --listen 127.0.0.1:8788 --path /webhooks/clover
```

`dev` starts a local webhook receiver and prints each verified JSON event as one
line. Set `CLOVER_WEBHOOK_SECRET` or pass `--secret`; the receiver verifies the
`X-Clover-Webhook-*` HMAC headers, rejects stale timestamps and duplicate
delivery IDs, and fails closed on malformed or oversized bodies. `stream` is a
compatibility alias for `dev`. The shared API client sends bearer auth,
user-agent, and idempotency headers, decodes problem JSON while retaining
unknown fields, and retries only safe requests or idempotent mutations with a
maximum of three attempts. Tests inject an HTTP round-tripper; no real network
is needed.

Build or test with `go test ./...` and install with
`go install github.com/openmirai/clover-sdks/apps/cli@latest`.
